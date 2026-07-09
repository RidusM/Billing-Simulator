package kafka

import (
	"context"
	"fmt"
	"time"

	"bill-stripe-sim/pkg/logger"

	"github.com/segmentio/kafka-go"
)

const (
	_defaultBatchSize    = 100
	_defaultBatchTimeout = 10 * time.Millisecond
)

type Producer struct {
	writer *kafka.Writer
	log    logger.Logger
	topic  string
}

func NewProducer(brokers []string, topic string, log logger.Logger) *Producer {
	staticMetadataCtx := context.WithValue(context.Background(), kafkaMetadataKey, map[string]string{
		"topic": topic,
	})

	return &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Balancer:     &kafka.Murmur2Balancer{},
			RequiredAcks: kafka.RequireAll,
			BatchSize:    _defaultBatchSize,
			BatchTimeout: _defaultBatchTimeout,
			Logger: kafka.LoggerFunc(func(msg string, args ...any) {
				var formatted string
				if len(args) > 0 {
					formatted = fmt.Sprintf(msg, args...)
				} else {
					formatted = msg
				}
				log.LogAttrs(staticMetadataCtx, logger.InfoLevel, "producer info",
					logger.String("message", formatted),
				)
			}),
			ErrorLogger: kafka.LoggerFunc(func(msg string, args ...any) {
				var formatted string
				if len(args) > 0 {
					formatted = fmt.Sprintf(msg, args...)
				} else {
					formatted = msg
				}
				log.LogAttrs(staticMetadataCtx, logger.ErrorLevel, "producer error",
					logger.String("error", formatted),
				)
			}),
			AllowAutoTopicCreation: false,
		},
		log:   log,
		topic: topic,
	}
}

func (p *Producer) Send(ctx context.Context, key, value []byte, headers ...kafka.Header) error {
	return p.SendToTopic(ctx, p.topic, key, value, headers...)
}

func (p *Producer) SendToTopic(ctx context.Context, topic string, key, value []byte, headers ...kafka.Header) error {
	err := p.writer.WriteMessages(ctx, kafka.Message{
		Topic:   topic,
		Key:     key,
		Value:   value,
		Headers: headers,
	})
	if err != nil {
		return fmt.Errorf("kafka.Producer.SendToTopic (topic: %s): %w", topic, err)
	}
	return nil
}

func (p *Producer) Close() error {
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("producer close: %w", err)
	}
	return nil
}
