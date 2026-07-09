package service

import (
	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"
	"context"
	"encoding/json"
	"fmt"
)

type OutboxRepository interface {
	SaveBatch(ctx context.Context, events []*entity.OutboxEvent) error
}

type EventDispatcher struct {
	outbox OutboxRepository
	clock  VirtualClock
	log    logger.Logger
}

func NewEventDispatcher(
	outbox OutboxRepository,
	clock VirtualClock,
	log logger.Logger,
) *EventDispatcher {
	return &EventDispatcher{
		outbox: outbox,
		log:    log,
	}
}

func (d *EventDispatcher) Dispatch(ctx context.Context, events entity.DomainEvents) error {
	if len(events) == 0 {
		return nil
	}

	outboxEvents := make([]*entity.OutboxEvent, 0, len(events))
	for _, event := range events {
		// Convert domain event to DTO for serialization
		dto := d.toDTO(event)

		payload, err := json.Marshal(dto)
		if err != nil {
			d.log.Error("failed to marshal event DTO",
				"event_type", event.EventType(),
				"error", err,
			)
			continue
		}

		outboxEvent := entity.NewOutboxEvent(event, d.clock.Now(), payload)
		outboxEvents = append(outboxEvents, outboxEvent)
	}

	if len(outboxEvents) == 0 {
		return nil
	}

	if err := d.outbox.SaveBatch(ctx, outboxEvents); err != nil {
		return fmt.Errorf("save outbox events: %w", err)
	}

	d.log.Debug("dispatched events to outbox",
		"count", len(outboxEvents),
	)
	return nil
}

// toDTO converts domain events to DTOs for serialization
func (d *EventDispatcher) toDTO(event entity.DomainEvent) any {
	switch e := event.(type) {
	case entity.CustomerCreatedEvent:
		return CustomerCreatedEventDTO{
			CustomerID:    e.CustomerID,
			CustomerPubID: e.CustomerPubID,
			Email:         e.Email,
			Name:          e.Name,
			CreatedAt:     e.CreatedAt,
		}
	case entity.InvoiceCreatedEvent:
		return InvoiceCreatedEventDTO{
			InvoiceID:         e.InvoiceID,
			InvoicePubID:      e.InvoicePubID,
			CustomerID:        e.CustomerID,
			CustomerPubID:     e.CustomerPubID,
			SubscriptionID:    e.SubscriptionID,
			SubscriptionPubID: e.SubscriptionPubID,
			Amount:            e.Amount,
			Currency:          e.Currency,
			Status:            string(e.Status),
			CreatedAt:         e.CreatedAt,
		}
	case entity.InvoicePaidEvent:
		return InvoicePaidEventDTO{
			InvoiceID:      e.InvoiceID,
			InvoicePubID:   e.InvoicePubID,
			CustomerID:     e.CustomerID,
			CustomerPubID:  e.CustomerPubID,
			SubscriptionID: e.SubscriptionID,
			Amount:         e.Amount,
			Currency:       e.Currency,
			PaidAt:         e.PaidAt,
		}
	case entity.InvoicePaymentFailedEvent:
		return InvoicePaymentFailedEventDTO{
			InvoiceID:      e.InvoiceID,
			InvoicePubID:   e.InvoicePubID,
			CustomerID:     e.CustomerID,
			CustomerPubID:  e.CustomerPubID,
			SubscriptionID: e.SubscriptionID,
			Amount:         e.Amount,
			Currency:       e.Currency,
			ErrorCode:      e.ErrorCode,
			FailedAt:       e.FailedAt,
		}
	case entity.SubscriptionCreatedEvent:
		return SubscriptionCreatedEventDTO{
			SubscriptionID:    e.SubscriptionID,
			SubscriptionPubID: e.SubscriptionPubID,
			CustomerID:        e.CustomerID,
			CustomerPubID:     e.CustomerPubID,
			PriceID:           e.PriceID,
			PricePubID:        e.PricePubID,
			Status:            string(e.Status),
			CurrentPeriodEnd:  e.CurrentPeriodEnd,
			NextBillingAt:     e.NextBillingAt,
			CreatedAt:         e.CreatedAt,
		}
	case entity.SubscriptionCanceledEvent:
		return SubscriptionCanceledEventDTO{
			SubscriptionID:    e.SubscriptionID,
			SubscriptionPubID: e.SubscriptionPubID,
			CustomerID:        e.CustomerID,
			CustomerPubID:     e.CustomerPubID,
			Status:            string(e.Status),
			CanceledAt:        e.CanceledAt,
			AtPeriodEnd:       e.AtPeriodEnd,
		}
	case entity.SubscriptionRenewedEvent:
		return SubscriptionRenewedEventDTO{
			SubscriptionID:    e.SubscriptionID,
			SubscriptionPubID: e.SubscriptionPubID,
			CustomerID:        e.CustomerID,
			CustomerPubID:     e.CustomerPubID,
			InvoiceID:         e.InvoiceID,
			InvoicePubID:      e.InvoicePubID,
			InvoiceAmount:     e.InvoiceAmount,
			InvoiceCurrency:   e.InvoiceCurrency,
			InvoiceStatus:     string(e.InvoiceStatus),
			NewPeriodEnd:      e.NewPeriodEnd,
			RenewedAt:         e.RenewedAt,
		}
	default:
		// Fallback: serialize domain event directly (should not happen)
		d.log.Warn("unknown domain event type, serializing directly",
			"event_type", event.EventType(),
		)
		return event
	}
}
