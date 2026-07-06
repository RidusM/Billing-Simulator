// internal/service/notification.go
package service

import (
	"bill-stripe-sim/internal/entity"
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

type EventSender interface {
	SendInvoiceEvent(ctx context.Context, event *InvoiceEvent) error
	SendSubscriptionEvent(ctx context.Context, event *SubscriptionEvent) error
}

type WebhookDispatcher interface {
	Deliver(ctx context.Context, customerID uuid.UUID, eventType entity.EventType, payload []byte) error
}

type NotificationService struct {
	sender  EventSender       // Kafka (внутренние события)
	webhook WebhookDispatcher // HTTP (доставка пользователю)
}

func NewNotificationService(sender EventSender, webhook WebhookDispatcher) *NotificationService {
	return &NotificationService{
		sender:  sender,
		webhook: webhook,
	}
}

func (s *NotificationService) NotifyInvoiceCreated(ctx context.Context, inv *entity.Invoice) error {
	event := mapInvoiceToEvent(inv)

	// 1. Внутреннее событие в Kafka
	if err := s.sender.SendInvoiceEvent(ctx, event); err != nil {
		return err
	}

	// 2. HTTP-доставка пользователю (асинхронно внутри Deliver)
	payload, _ := json.Marshal(event)
	_ = s.webhook.Deliver(ctx, inv.CustomerID, entity.EventInvoiceCreated, payload)

	return nil
}

func (s *NotificationService) NotifySubscriptionUpdated(ctx context.Context, sub *entity.Subscription) error {
	event := mapSubscriptionToEvent(sub)

	if err := s.sender.SendSubscriptionEvent(ctx, event); err != nil {
		return err
	}

	payload, _ := json.Marshal(event)
	_ = s.webhook.Deliver(ctx, sub.CustomerID, entity.EventSubscriptionRenewed, payload)

	return nil
}
