package dlq

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"bill-stripe-sim/pkg/logger"

	"github.com/segmentio/kafka-go"
)

type Publisher interface {
	SendToTopic(ctx context.Context, topic string, key, value []byte, headers ...kafka.Header) error
}

type dlqMessage struct {
	msg     kafka.Message
	err     error
	attempt int
}

type DLQ struct {
    producer Publisher
    topic    string
    logger   logger.Logger
    ch       chan dlqMessage
    wg       sync.WaitGroup
    cancel   context.CancelFunc
    cfg      *config
}

func New(producer Publisher, topic string, logger logger.Logger, opts ...Option) *DLQ {
    cfg := defaultConfig()
    for _, opt := range opts {
        opt(cfg)
    }

    ctx, cancel := context.WithCancel(context.Background())
    d := &DLQ{
        producer: producer,
        topic:    topic,
        logger:   logger,
        ch:       make(chan dlqMessage, cfg.BufferSize),
        cancel:   cancel,
        cfg:      cfg,
    }

    d.wg.Add(1)
    go d.worker(ctx)

    return d
}

func (d *DLQ) PublishError(ctx context.Context, msg kafka.Message, err error, attempt int) error {
    m := dlqMessage{msg: msg, err: err, attempt: attempt}

    select {
    case d.ch <- m:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    default:
        d.logger.LogAttrs(ctx, logger.WarnLevel, "dlq buffer full, falling back to sync send",
            logger.String("topic", d.topic),
        )
        return d.send(context.Background(), m)
    }
}

func (d *DLQ) worker(ctx context.Context) {
    defer d.wg.Done()

    backoff := d.cfg.InitialBackoff

    for {
        select {
        case <-ctx.Done():
            for m := range d.ch {
                if err := d.send(context.Background(), m); err != nil {
                    d.logger.LogAttrs(context.Background(), logger.ErrorLevel, "failed to send dlq message during shutdown",
                        logger.Any("error", err),
                    )
                }
            }
            return
        case m := <-d.ch:
            if err := d.send(ctx, m); err != nil {
                d.logger.LogAttrs(ctx, logger.ErrorLevel, "failed to send dlq message, re-enqueueing",
                    logger.String("topic", d.topic),
                    logger.Any("error", err),
                    logger.Duration("backoff", backoff),
                )
                select {
                case <-time.After(backoff):
                    backoff = min(backoff*2, d.cfg.MaxBackoff)

                    select {
                    case d.ch <- m:
                    default:
                        d.logger.LogAttrs(ctx, logger.ErrorLevel, "dlq buffer full after backoff, message lost",
                            logger.Any("error", err),
                        )
                    }
                case <-ctx.Done():
                    return
                }
            } else {
                backoff = d.cfg.InitialBackoff
            }
        }
    }
}

func (d *DLQ) send(ctx context.Context, m dlqMessage) error {
    const op = "kafka.dlq.PublishError"

    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    payload := map[string]any{
        "original_topic": m.msg.Topic,
        "error":          m.err.Error(),
        "attempt":        m.attempt,
        "timestamp":      time.Now().UTC(),
        "data_base64":    base64.StdEncoding.EncodeToString(m.msg.Value),
    }

    val, errMarshal := json.Marshal(payload)
    if errMarshal != nil {
        d.logger.LogAttrs(ctx, logger.ErrorLevel, "failed to marshal dlq payload",
            logger.String("op", op),
            logger.Any("err", errMarshal),
        )
        val = []byte(fmt.Sprintf(`{"status":"marshal_error","raw_data":"%s","error":"%s"}`,
            string(m.msg.Value), m.err.Error()))
    }

    if errSend := d.producer.SendToTopic(ctx, d.topic, m.msg.Key, val, m.msg.Headers...); errSend != nil {
        return fmt.Errorf("%s: send to kafka: %w", op, errSend)
    }
    return nil
}

func (d *DLQ) Close() error {
    d.cancel()
    close(d.ch)
    d.wg.Wait()
    return nil
}

func (d *DLQ) Topic() string { return d.topic }