// internal/service/webhook.go
package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"

	"github.com/google/uuid"
)

// WebhookDeliveryService — движок доставки вебхуков пользователю.
type WebhookDeliveryService struct {
	endpoints WebhookEndpointRepository // найти, куда слать
	logs      WebhookLogRepository      // записать попытку
	events    EventRepository           // audit log
	publisher EventPublisher            // Kafka DLQ после max retries
	client    HTTPClient                // абстракция над http.Client
	clock     TimeProvider              // VirtualClock для timestamp подписи
	log       logger.Logger

	cfg WebhookConfig
}

type WebhookConfig struct {
	MaxRetries     int           // 5
	InitialBackoff time.Duration // 1s
	MaxBackoff     time.Duration // 5min
	RequestTimeout time.Duration // 10s
	DLQTopic       string        // "billing.dlq.webhooks"
}

// Deliver — главная точка входа. Вызывается из NotificationService.
// Асинхронна: не блокирует вызывающий код.
func (s *WebhookDeliveryService) Deliver(ctx context.Context, customerID uuid.UUID, eventType entity.EventType, payload []byte) error {
	// 1. Найти активные endpoints для customer
	endpoints, err := s.endpoints.GetActiveByCustomerID(ctx, customerID)
	if err != nil {
		return err
	}
	if len(endpoints) == 0 {
		return nil // у customer нет webhook endpoints — ОК
	}

	// 2. Создать trace_id для этой доставки
	traceID := uuid.New()

	// 3. Для каждого endpoint — отправить асинхронно
	for _, ep := range endpoints {
		go s.deliverToEndpoint(context.Background(), ep, traceID, eventType, payload)
	}

	return nil
}

// deliverToEndpoint — отправка одному endpoint с retry.
func (s *WebhookDeliveryService) deliverToEndpoint(
	ctx context.Context,
	ep *entity.WebhookEndpoint,
	traceID uuid.UUID,
	eventType entity.EventType,
	payload []byte,
) {
	// Подписать payload
	timestamp := s.clock.Now().Unix()
	signature := ep.SignPayload(payload, timestamp)

	backoff := s.cfg.InitialBackoff

	for attempt := 1; attempt <= s.cfg.MaxRetries; attempt++ {
		// Записать попытку в webhook_logs
		logEntry := &entity.WebhookLog{
			ID:        uuid.New(),
			TraceID:   traceID,
			EventType: string(eventType),
			Payload:   payload,
			TargetURL: ep.URL,
			Status:    entity.WebhookStatusPending,
			Attempt:   attempt,
			CreatedAt: s.clock.Now(),
		}
		_ = s.logs.Create(ctx, logEntry)

		// Отправить HTTP POST
		statusCode, err := s.sendHTTP(ctx, ep.URL, payload, signature, timestamp)

		if err == nil && statusCode >= 200 && statusCode < 300 {
			// Успех
			logEntry.Status = entity.WebhookStatusDelivered
			logEntry.ResponseCode = &statusCode
			_ = s.logs.Update(ctx, logEntry)

			// Audit event
			event, _ := entity.NewEvent(entity.EventWebhookDelivered, map[string]any{
				"webhook_log_id": logEntry.ID,
				"target_url":     ep.URL,
				"status_code":    statusCode,
			})
			_ = s.events.Create(ctx, event)
			return
		}

		// Неудача — обновить лог
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

		// Если это последняя попытка — в DLQ
		if attempt == s.cfg.MaxRetries {
			_ = s.publisher.Publish(ctx, s.cfg.DLQTopic, payload, map[string]any{
				"trace_id":   traceID,
				"endpoint":   ep.URL,
				"event_type": eventType,
				"attempts":   attempt,
				"error":      errMsg,
			})
			return
		}

		// Ждём backoff
		time.Sleep(backoff)
		backoff = min(backoff*2, s.cfg.MaxBackoff)
	}
}

func (s *WebhookDeliveryService) sendHTTP(
	ctx context.Context,
	url string,
	payload []byte,
	signature string,
	timestamp int64,
) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.RequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Simulator-Signature", fmt.Sprintf("t=%d,v1=%s", timestamp, signature))
	req.Header.Set("User-Agent", "BillingStripeSim/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func (s *WebhookDeliveryService) QueueWebhook(ctx context.Context, ep *entity.WebhookEndpoint, payload []byte) {
	// Создаем контекст, который не умрет после завершения HTTP-ответа
	detachedCtx := context.WithoutCancel(ctx)

	// Добавляем жесткий таймаут непосредственно на операцию сетевой доставки
	deliveryCtx, cancel := context.WithTimeout(detachedCtx, 30*time.Second)

	go func() {
		defer cancel()
		s.deliverToEndpoint(deliveryCtx, ep, traceID, eventType, payload)
	}()
}
