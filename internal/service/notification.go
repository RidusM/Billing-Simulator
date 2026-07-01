package service

import (
	"bill-stripe-sim/internal/entity"
	"context"
)

type EventSender interface {
	SendInvoiceEvent(ctx context.Context, invoice *entity.Invoice) error
	SendSubscriptionEvent(ctx context.Context, sub *entity.Subscription) error
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
	return s.sender.SendInvoiceEvent(ctx, inv)
}

func (s *NotificationService) NotifySubscriptionUpdated(ctx context.Context, sub *entity.Subscription) error {
	return s.sender.SendSubscriptionEvent(ctx, sub)
}
