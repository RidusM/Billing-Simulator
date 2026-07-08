package service

import (
	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"
	"context"
	"encoding/json"
	"fmt"
)

// OutboxRepository — интерфейс объявлен ЗДЕСЬ, в service пакете
type OutboxRepository interface {
	SaveBatch(ctx context.Context, events []*entity.OutboxEvent) error
}

type EventDispatcher struct {
	outbox OutboxRepository // ← Локальный интерфейс!
	clock  VirtualClock
	log    logger.Logger
}

func NewEventDispatcher(
	outbox OutboxRepository, // ← Локальный интерфейс!
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
		payload, err := json.Marshal(event)
		if err != nil {
			d.log.Error("failed to marshal event",
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
