package kafka

import (
	"bill-stripe-sim/internal/service" // Импортируем service вместо entity
	"bill-stripe-sim/pkg/kafka"
	"context"
	"encoding/json"
	"fmt"
)

type EventSender struct {
	producer          *kafka.Producer
	invoiceTopic      string
	subscriptionTopic string
}

func NewEventSender(producer *kafka.Producer, invTopic, subTopic string) *EventSender {
	return &EventSender{
		producer:          producer,
		invoiceTopic:      invTopic,
		subscriptionTopic: subTopic,
	}
}

func (s *EventSender) SendInvoiceEvent(ctx context.Context, inv *service.InvoiceEvent) error {

	payload, err := json.Marshal(inv)
	if err != nil {
		return fmt.Errorf("marshal invoice event: %w", err)
	}

	return s.producer.SendToTopic(ctx, s.invoiceTopic, []byte(inv.ID.String()), payload)
}

func (s *EventSender) SendSubscriptionEvent(ctx context.Context, sub *service.SubscriptionEvent) error {
	payload, err := json.Marshal(sub)
	if err != nil {
		return fmt.Errorf("marshal subscription event: %w", err)
	}

	return s.producer.SendToTopic(ctx, s.subscriptionTopic, []byte(sub.ID.String()), payload)
}
