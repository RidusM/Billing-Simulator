package service

import (
	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"
	"context"
	"time"

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

type WebhookSender interface {
	Send(ctx context.Context, url string, payload []byte, signature string, timestamp int64) (int, error)
}

type WebhookDeliveryService struct {
	endpoints WebhookEndpointRepository
	logs      WebhookLogRepository
	events    EventRepository
	sender    WebhookSender
	clock     VirtualClock
	log       logger.Logger
}

func NewWebhookDeliveryService(
	endpoints WebhookEndpointRepository,
	logs WebhookLogRepository,
	events EventRepository,
	sender WebhookSender,
	clock VirtualClock,
	log logger.Logger,
) *WebhookDeliveryService {
	return &WebhookDeliveryService{
		endpoints: endpoints,
		logs:      logs,
		events:    events,
		sender:    sender,
		clock:     clock,
		log:       log,
	}
}

func (s *WebhookDeliveryService) Deliver(ctx context.Context, customerID uuid.UUID, eventType string, payload []byte) error {
	endpoints, err := s.endpoints.GetActiveByCustomerID(ctx, customerID)
	if err != nil {
		return err
	}
	if len(endpoints) == 0 {
		return nil
	}

	traceID := uuid.New()

	sem := make(chan struct{}, 10)

	for _, ep := range endpoints {
		sem <- struct{}{}

		go func(endpoint *entity.WebhookEndpoint) {
			defer func() { <-sem }()

			s.deliverToEndpoint(context.WithoutCancel(ctx), endpoint, traceID, eventType, payload)
		}(ep)
	}

	return nil
}

func (s *WebhookDeliveryService) deliverToEndpoint(
	ctx context.Context,
	ep *entity.WebhookEndpoint,
	traceID uuid.UUID,
	eventType string,
	payload []byte,
) {
	now := s.clock.Now()
	timestamp := now.Unix()
	signature := ep.SignPayload(payload, timestamp)

	event := &entity.Event{
		ID:        uuid.New(),
		PublicID:  "evt_" + uuid.New().String()[:8],
		Type:      entity.EventType(eventType),
		Payload:   payload,
		CreatedAt: now,
	}
	if err := s.events.Create(ctx, event); err != nil {
		s.log.Error("failed to create event", "error", err)
		return
	}

	const maxRetries = 5
	const initialBackoff = 1 * time.Second
	const maxBackoff = 5 * time.Minute

	backoff := initialBackoff

	for attempt := 1; attempt <= maxRetries; attempt++ {
		currentNow := s.clock.Now()
		logEntry := &entity.WebhookLog{
			ID:         uuid.New(),
			PublicID:   "wl_" + uuid.New().String()[:8],
			EventID:    event.ID,
			EndpointID: ep.ID,
			TraceID:    traceID,
			EventType:  eventType,
			Payload:    payload,
			TargetURL:  ep.URL,
			Status:     entity.WebhookStatusPending,
			Attempt:    attempt,
			CreatedAt:  currentNow,
			UpdatedAt:  currentNow,
		}

		if err := s.logs.Create(ctx, logEntry); err != nil {
			s.log.Error("failed to create webhook log", "error", err)
		}

		statusCode, err := s.sender.Send(ctx, ep.URL, payload, signature, timestamp)
		if err == nil && statusCode >= 200 && statusCode < 300 {
			logEntry.Status = entity.WebhookStatusDelivered
			logEntry.ResponseCode = &statusCode
			logEntry.UpdatedAt = s.clock.Now()
			_ = s.logs.Update(ctx, logEntry)
			return
		}

		if err == nil && statusCode >= 200 && statusCode < 300 {
			logEntry.Status = entity.WebhookStatusDelivered
			logEntry.ResponseCode = &statusCode
			logEntry.UpdatedAt = s.clock.Now()
			_ = s.logs.Update(ctx, logEntry)

			event := &entity.Event{
				ID:        uuid.New(),
				PublicID:  "evt_" + uuid.New().String()[:8],
				Type:      "webhook.delivered",
				Payload:   payload,
				CreatedAt: s.clock.Now(),
			}
			_ = s.events.Create(ctx, event)
			return
		}

		errMsg := "request failed"
		if err != nil {
			errMsg = err.Error()
		}
		logEntry.Status = entity.WebhookStatusFailed
		if statusCode > 0 {
			logEntry.ResponseCode = &statusCode
		}
		logEntry.ErrorMessage = &errMsg
		logEntry.UpdatedAt = s.clock.Now()
		_ = s.logs.Update(ctx, logEntry)

		if attempt == maxRetries {
			return
		}

		time.Sleep(backoff)
		backoff = min(backoff*2, maxBackoff)
	}
}
