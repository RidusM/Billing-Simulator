package kafka

import (
	"context"
	"fmt"

	"bill-stripe-sim/pkg/logger"

	"github.com/segmentio/kafka-go"
)

type contextKey string

const kafkaMetadataKey contextKey = "kafka_metadata"

type Consumer struct {
	reader *kafka.Reader
	log    logger.Logger
}

func NewConsumer(brokers []string, topic, groupID string, log logger.Logger) *Consumer {
	staticMetadataCtx := context.WithValue(context.Background(), kafkaMetadataKey, map[string]string{
		"topic":    topic,
		"group_id": groupID,
	})

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: groupID,
		Logger: kafka.LoggerFunc(func(msg string, args ...any) {
			log.LogAttrs(staticMetadataCtx, logger.Info, "consumer info",
				logger.String("message", fmt.Sprintf(msg, args...)),
			)
		}),
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...any) {
			log.LogAttrs(staticMetadataCtx, logger.Error, "consumer error",
				logger.String("error", fmt.Sprintf(msg, args...)),
			)
		}),
	})

	return &Consumer{
		reader: reader,
		log:    log,
	}
}

func (c *Consumer) Fetch(ctx context.Context) (kafka.Message, error) {
	msg, err := c.reader.FetchMessage(ctx)
	if err != nil {
		return kafka.Message{}, fmt.Errorf("kafka.Consumer.Fetch: %w", err)
	}
	return msg, nil
}

func (c *Consumer) Commit(ctx context.Context, msg kafka.Message) error {
	if err := c.reader.CommitMessages(ctx, msg); err != nil {
		return fmt.Errorf("kafka.Consumer.Commit: %w", err)
	}
	return nil
}

func (c *Consumer) Close() error {
	if err := c.reader.Close(); err != nil {
		return fmt.Errorf("consumer close: %w", err)
	}
	return nil
}
