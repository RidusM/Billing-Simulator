package service

import (
	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"
	"context"

	"github.com/google/uuid"
)

type WebhookEndpointRepository interface {
	GetActiveByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*entity.WebhookEndpoint, error)
}

type WebhookLogRepository interface {
	Create(ctx context.Context, wl *entity.WebhookLog) error
	Update(ctx context.Context, wl *entity.WebhookLog) error
}

type EventRepository interface {
	Create(ctx context.Context, e *entity.Event) error
}

type WebhookDeliveryService struct {
	endpoints WebhookEndpointRepository
	logs      WebhookLogRepository
	events    EventRepository
	clock     VirtualClock
	log       logger.Logger
}

func NewWebhookDeliveryService(
	endpoints WebhookEndpointRepository,
	logs WebhookLogRepository,
	events EventRepository,
	clock VirtualClock,
	log logger.Logger,
) *WebhookDeliveryService {
	return &WebhookDeliveryService{
		endpoints: endpoints,
		logs:      logs,
		events:    events,
		clock:     clock,
		log:       log,
	}
}

// Deliver — создаёт WebhookLog записи в БД для обработки WebhookRetryWorker
func (s *WebhookDeliveryService) Deliver(ctx context.Context, customerID uuid.UUID, eventType string, payload []byte) error {
	endpoints, err := s.endpoints.GetActiveByCustomerID(ctx, customerID)
	if err != nil {
		return err
	}
	if len(endpoints) == 0 {
		return nil
	}

	traceID := uuid.New()
	now := s.clock.Now()

	for _, ep := range endpoints {
		// 1. Создать Event запись
		event := &entity.Event{
			ID:        uuid.New(),
			PublicID:  "evt_" + uuid.New().String()[:8],
			Type:      entity.EventType(eventType),
			Payload:   payload,
			CreatedAt: now,
		}
		if err := s.events.Create(ctx, event); err != nil {
			s.log.Error("failed to create event", "error", err)
			continue
		}

		// 2. Создать WebhookLog запись для retry worker
		wl, err := entity.NewWebhookLog(event.ID, ep.ID, traceID, eventType, payload, ep.URL, now)
		if err != nil {
			s.log.Error("failed to create webhook log", "error", err)
			continue
		}

		if err := s.logs.Create(ctx, wl); err != nil {
			s.log.Error("failed to save webhook log", "error", err)
			continue
		}

		s.log.Debug("webhook log created, scheduled for delivery",
			"webhook_log_id", wl.PublicID,
			"endpoint_url", ep.URL,
		)
	}

	return nil
}
