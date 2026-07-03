package kafka

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"bill-stripe-sim/pkg/kafka/dlq"
	"bill-stripe-sim/pkg/logger"

	"github.com/segmentio/kafka-go"
)

const _backoffMultiplier = 2

type Handler func(ctx context.Context, msg kafka.Message) error

type Processor struct {
	consumer *Consumer
	dlq      *dlq.DLQ
	logger   logger.Logger

	cfg Config
	wg  sync.WaitGroup
	cancel context.CancelFunc
}

func NewProcessor(
	c *Consumer,
	d *dlq.DLQ,
	logger logger.Logger,
	opts ...ProcessorOption,
) (*Processor, error) {
	if d == nil {
        return nil, fmt.Errorf("kafka.NewProcessor: DLQ is required for guaranteed delivery")
    }

	cfg := DefaultConfig()

	for _, opt := range opts {
		opt(&cfg)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("kafka.NewProcessor: validation: %w", err)
	}

	return &Processor{
		consumer: c,
		dlq:      d,
		logger:   logger,
		cfg:      cfg,
	}, nil
}

func (p *Processor) Start(ctx context.Context, handler Handler) {
	ctx, p.cancel = context.WithCancel(ctx)
	for range p.cfg.WorkersCount {
		p.wg.Add(1)
		go p.worker(ctx, handler)
	}
}

func (p *Processor) Stop() {
	if p.cancel != nil {
        p.cancel()
    }
    p.consumer.Close()
    p.wg.Wait()
}

func (p *Processor) worker(ctx context.Context, handler Handler) {
	defer p.wg.Done()

	for {
		msg, err := p.consumer.Fetch(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			p.logger.LogAttrs(ctx, logger.ErrorLevel, "fetch error",
				logger.Any("error", err),
			)
			continue
		}

		p.processWithRetry(ctx, msg, handler)
	}
}

func (p *Processor) processWithRetry(ctx context.Context, msg kafka.Message, handler Handler) {
	var lastErr error

	currentBackoff := p.cfg.BaseRetryDelay

	for attempt := 1; attempt <= p.cfg.MaxAttempts; attempt++ {
		lastErr = handler(ctx, msg)
		if lastErr == nil {
			if err := p.consumer.Commit(ctx, msg); err != nil {
				p.logger.LogAttrs(ctx, logger.ErrorLevel, "failed to commit message offset",
					logger.Int64("offset", msg.Offset),
					logger.String("topic", msg.Topic),
					logger.Any("error", err),
				)
			}
			return
		}

		if IsUnretryable(lastErr) {
			p.logger.LogAttrs(ctx, logger.ErrorLevel, "unretryable error, sending to DLQ",
				logger.String("topic", msg.Topic),
				logger.Int64("offset", msg.Offset),
				logger.Any("error", lastErr),
			)
			break
		}

		p.logger.LogAttrs(ctx, logger.WarnLevel, "retryable error",
			logger.Int("attempt", attempt),
			logger.Any("err", lastErr),
		)

		if attempt >= p.cfg.MaxAttempts {
			break
		}

		//nolint:gosec // weak random is completely fine for exponential backoff jitter
		jitter := min(time.Duration(
			rand.Int64N(int64(currentBackoff*_backoffMultiplier)),
		), p.cfg.MaxRetryDelay)

		select {
		case <-time.After(jitter):
		case <-ctx.Done():
			return
		}

		if currentBackoff < p.cfg.MaxRetryDelay/2 {
    		currentBackoff *= _backoffMultiplier
} else {
    currentBackoff = p.cfg.MaxRetryDelay
}
	}

	if err := p.dlq.PublishError(ctx, msg, lastErr, p.cfg.MaxAttempts); err != nil {
    p.logger.LogAttrs(ctx, logger.ErrorLevel, "dlq publish failed (buffer full + sync fallback failed)",
        logger.Any("err", err),
    )
    return
}

	if err := p.consumer.Commit(ctx, msg); err != nil {
		p.logger.LogAttrs(ctx, logger.ErrorLevel, "final commit error",
			logger.Any("err", err),
		)
	}
}
