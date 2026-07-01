package kafka

import (
	"bill-stripe-sim/pkg/logger"
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
	log    logger.Logger
	topic  string
}

func NewProducer(brokers []string, topic string, log logger.Logger) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Balancer:     &kafka.Murmur2Balancer{},
			RequiredAcks: kafka.RequireAll,
			// Настройки для Highload: батчинг сообщений
			BatchSize:    100,
			BatchTimeout: 10 * time.Millisecond,
			Logger: kafka.LoggerFunc(func(msg string, args ...any) {
				log.LogAttrs(context.Background(), logger.Info, "producer info",
					logger.String("message", fmt.Sprintf(msg, args...)),
				)
			}),
			ErrorLogger: kafka.LoggerFunc(func(msg string, args ...any) {
				log.LogAttrs(context.Background(), logger.Error, "producer error",
					logger.String("error", fmt.Sprintf(msg, args...)),
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
