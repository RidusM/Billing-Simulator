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
	sender    WebhookSender // ← HTTP вынесен в transport layer
	clock     TimeProvider
	log       logger.Logger
}

func NewWebhookDeliveryService(
	endpoints WebhookEndpointRepository,
	logs WebhookLogRepository,
	events EventRepository,
	sender WebhookSender,
	clock TimeProvider,
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

	for _, ep := range endpoints {
		go s.deliverToEndpoint(context.Background(), ep, traceID, eventType, payload)
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
	timestamp := s.clock.Now().Unix()
	signature := ep.SignPayload(payload, timestamp)

	// Константы вместо конфига
	const maxRetries = 5
	const initialBackoff = 1 * time.Second
	const maxBackoff = 5 * time.Minute

	backoff := initialBackoff

	for attempt := 1; attempt <= maxRetries; attempt++ {
		logEntry := &entity.WebhookLog{
			ID:        uuid.New(),
			TraceID:   traceID,
			EventType: eventType,
			Payload:   payload,
			TargetURL: ep.URL,
			Status:    entity.WebhookStatusPending,
			Attempt:   attempt,
			CreatedAt: s.clock.Now(),
		}
		_ = s.logs.Create(ctx, logEntry)

		statusCode, err := s.sender.Send(ctx, ep.URL, payload, signature, timestamp)

		if err == nil && statusCode >= 200 && statusCode < 300 {
			logEntry.Status = entity.WebhookStatusDelivered
			logEntry.ResponseCode = &statusCode
			_ = s.logs.Update(ctx, logEntry)

			event, _ := entity.NewEvent(entity.EventWebhookDelivered, map[string]any{
				"webhook_log_id": logEntry.ID,
				"target_url":     ep.URL,
				"status_code":    statusCode,
			})
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
		_ = s.logs.Update(ctx, logEntry)

		if attempt == maxRetries {
			return
		}

		time.Sleep(backoff)
		backoff = min(backoff*2, maxBackoff)
	}
}
