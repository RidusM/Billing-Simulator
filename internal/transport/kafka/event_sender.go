package kafkatransport

import (
	"context"
	"encoding/json"
	"fmt"

	"bill-stripe-sim/pkg/logger"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
)

// Producer — то немногое, что нужно от kafka.Producer.
// Объявлен здесь (в transport/kafka), а не переиспользуем *kafka.Producer напрямую,
// чтобы не тащить конкретный тип в service-слой и не усложнять моки в тестах.
type Producer interface {
	SendToTopic(ctx context.Context, topic string, key, value []byte, headers ...kafkago.Header) error
}

// EventSenderAdapter реализует service.EventSender поверх kafka.Producer.
//
// ВАЖНО: сигнатуры не совпадают один-в-один:
//   - service.EventSender.Send(ctx, topic, payload, headers map[string]string)
//   - kafka.Producer.Send(ctx, key, value, headers ...kafka.Header) — топик фиксирован в Producer
//
// Поэтому используем SendToTopic и сами конвертируем headers + вычисляем partition key.
type EventSenderAdapter struct {
	producer Producer
	log      logger.Logger
}

func NewEventSenderAdapter(producer Producer, log logger.Logger) *EventSenderAdapter {
	return &EventSenderAdapter{producer: producer, log: log}
}

// Send — реализация service.EventSender.
func (a *EventSenderAdapter) Send(ctx context.Context, topic string, payload []byte, headers map[string]string) error {
	const op = "kafkatransport.EventSenderAdapter.Send"

	kHeaders := make([]kafkago.Header, 0, len(headers))
	for k, v := range headers {
		kHeaders = append(kHeaders, kafkago.Header{Key: k, Value: []byte(v)})
	}

	key := partitionKey(payload)

	if err := a.producer.SendToTopic(ctx, topic, key, payload, kHeaders...); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// partitionKey — пытаемся сохранить порядок доставки событий одного customer'а в рамках партиции
// (важно для дашборда: "created" не должен догнать "renewed" из-за реордеринга в разных партициях).
// Не все domain-события несут customer_id в payload (например EventWebhookDelivered) — в этом
// случае используем случайный ключ, порядок для таких событий не критичен.
func partitionKey(payload []byte) []byte {
	var probe struct {
		CustomerID string `json:"customer_id"`
	}
	if err := json.Unmarshal(payload, &probe); err == nil && probe.CustomerID != "" {
		return []byte(probe.CustomerID)
	}
	return []byte(uuid.NewString())
}
