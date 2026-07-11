package service

import (
	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type EventSender interface {
	Send(ctx context.Context, topic string, payload []byte, headers map[string]string) error
}

type WebhookDispatcher interface {
	Deliver(ctx context.Context, customerID uuid.UUID, eventType string, payload []byte) error
}

type NotificationService struct {
	sender  EventSender
	webhook WebhookDispatcher
	log     logger.Logger
}

func NewNotificationService(
	sender EventSender,
	webhook WebhookDispatcher,
	log logger.Logger,
) *NotificationService {
	return &NotificationService{
		sender:  sender,
		webhook: webhook,
		log:     log,
	}
}

// HandleDomainEvent — принимает УЖЕ СЕРИАЛИЗОВАННЫЙ payload
func (s *NotificationService) HandleDomainEvent(
	ctx context.Context,
	eventType string,
	customerID uuid.UUID,
	payload []byte,
) error {
	switch eventType {
	case "customer.created":
		return s.handleEvent(ctx, customerID, entity.EventCustomerCreated, payload)

	case "subscription.created":
		return s.handleEvent(ctx, customerID, entity.EventSubscriptionCreated, payload)

	case "subscription.canceled":
		return s.handleEvent(ctx, customerID, entity.EventSubscriptionCanceled, payload)

	case "subscription.renewed":
		return s.handleEvent(ctx, customerID, entity.EventSubscriptionRenewed, payload)

	case "invoice.created":
		return s.handleEvent(ctx, customerID, entity.EventInvoiceCreated, payload)

	case "invoice.paid":
		return s.handleEvent(ctx, customerID, entity.EventInvoicePaid, payload)

	case "invoice.payment_failed":
		return s.handleEvent(ctx, customerID, entity.EventInvoicePaymentFailed, payload)

	default:
		s.log.Warn("unknown domain event type", "event_type", eventType)
		return nil
	}
}

func (s *NotificationService) handleEvent(ctx context.Context, customerID uuid.UUID, eventType entity.EventType, payload []byte) error {
	var multiErr error

	// Kafka
	topic := fmt.Sprintf("billing.%s", string(eventType))
	if err := s.sender.Send(ctx, topic, payload, nil); err != nil {
		s.log.Error("failed to send to Kafka", "error", err)
		multiErr = fmt.Errorf("kafka: %w", err)
	}

	// Webhook
	if err := s.webhook.Deliver(ctx, customerID, string(eventType), payload); err != nil {
		s.log.Error("failed to deliver webhook", "error", err)
		if multiErr != nil {
			multiErr = fmt.Errorf("%v; webhook: %w", multiErr, err)
		} else {
			multiErr = fmt.Errorf("webhook: %w", err)
		}
	}

	return multiErr
}
