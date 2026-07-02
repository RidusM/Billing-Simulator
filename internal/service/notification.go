package service

import (
	"bill-stripe-sim/internal/entity"
	"context"
)

type EventSender interface {
	SendInvoiceEvent(ctx context.Context, event *InvoiceEvent) error
	SendSubscriptionEvent(ctx context.Context, event *SubscriptionEvent) error
}

type NotificationService struct {
	sender EventSender
}

func NewNotificationService(sender EventSender) *NotificationService {
	return &NotificationService{
		sender: sender,
	}
}

func (s *NotificationService) NotifyInvoiceCreated(ctx context.Context, inv *entity.Invoice) error {
	event := mapInvoiceToEvent(inv)
	return s.sender.SendInvoiceEvent(ctx, event)
}

func (s *NotificationService) NotifySubscriptionUpdated(ctx context.Context, sub *entity.Subscription) error {
	event := mapSubscriptionToEvent(sub)
	return s.sender.SendSubscriptionEvent(ctx, event)
}
