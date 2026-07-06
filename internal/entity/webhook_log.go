package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type WebhookStatus string

const (
	WebhookStatusPending   WebhookStatus = "pending"
	WebhookStatusDelivered WebhookStatus = "delivered"
	WebhookStatusFailed    WebhookStatus = "failed"
)

type WebhookLog struct {
	ID            uuid.UUID
	TraceID       uuid.UUID
	EventType     string
	Payload       json.RawMessage
	TargetURL     string
	Status        WebhookStatus
	ResponseCode  *int
	Attempt       int
	ErrorMessage  *string
	NextAttemptAt time.Time
	CreatedAt     time.Time
}

func NewWebhookLog(traceID uuid.UUID, eventType string, payload json.RawMessage, targetURL string, now time.Time) *WebhookLog {
	return &WebhookLog{
		ID:            uuid.New(),
		TraceID:       traceID,
		EventType:     eventType,
		Payload:       payload,
		TargetURL:     targetURL,
		Status:        WebhookStatusPending,
		Attempt:       1,
		NextAttemptAt: now.UTC(),
		CreatedAt:     now.UTC(),
	}
}

func (wl *WebhookLog) MarkDelivered(responseCode int) {
	wl.Status = WebhookStatusDelivered
	wl.ResponseCode = &responseCode
	wl.ErrorMessage = nil
}
