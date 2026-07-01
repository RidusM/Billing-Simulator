package dlq

import (
	"bill-stripe-sim/pkg/logger"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

type Publisher interface {
	SendToTopic(ctx context.Context, topic string, key, value []byte, headers ...kafka.Header) error
}

type DLQ struct {
	producer Publisher
	topic    string
	logger   logger.Logger
}

func New(producer Publisher, topic string, logger logger.Logger) *DLQ {
	return &DLQ{
		producer: producer,
		topic:    topic,
		logger:   logger,
	}
}

func (d *DLQ) PublishError(ctx context.Context, msg kafka.Message, err error, attempt int) error {
	const op = "kafka.dlq.PublishError"

	payload := map[string]any{
		"original_topic": msg.Topic,
		"error":          err.Error(),
		"attempt":        attempt,
		"timestamp":      time.Now().UTC(),
		"data_base64":    base64.StdEncoding.EncodeToString(msg.Value),
	}

	val, errMarshal := json.Marshal(payload)
	if errMarshal != nil {
		d.logger.LogAttrs(ctx, logger.Error, "failed to marshal dlq payload",
			logger.String("op", op),
			logger.Any("err", errMarshal),
		)

		fallbackData := fmt.Sprintf(`{"status":"marshal_error","raw_data":"%s","error":"%s"}`,
			string(msg.Value), err.Error())
		val = []byte(fallbackData)
	}

	if errSend := d.producer.SendToTopic(ctx, d.topic, msg.Key, val, msg.Headers...); errSend != nil {
		return fmt.Errorf("%s: send to kafka: %w", op, errSend)
	}

	return nil
}

func (d *DLQ) Topic() string {
	return d.topic
}
