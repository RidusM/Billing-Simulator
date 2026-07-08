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
	PublicID      string
	EventID       uuid.UUID
	EndpointID    uuid.UUID
	TraceID       uuid.UUID
	EventType     string
	Payload       json.RawMessage
	TargetURL     string
	Status        WebhookStatus
	ResponseCode  *int
	ResponseBody  string
	Attempt       int
	MaxAttempts   int
	ErrorMessage  *string
	NextAttemptAt time.Time
	DeliveredAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewWebhookLog(eventID, endpointID, traceID uuid.UUID, eventType string, payload json.RawMessage, targetURL string, now time.Time) *WebhookLog {
	pubID, _ := GeneratePublicID("wh")
	return &WebhookLog{
		ID:            uuid.New(),
		PublicID:      pubID,
		EventID:       eventID,
		EndpointID:    endpointID,
		TraceID:       traceID,
		EventType:     eventType,
		Payload:       payload,
		TargetURL:     targetURL,
		Status:        WebhookStatusPending,
		Attempt:       1,
		MaxAttempts:   5,
		NextAttemptAt: now.UTC(),
		CreatedAt:     now.UTC(),
		UpdatedAt:     now.UTC(),
	}
}

func (wl *WebhookLog) MarkDelivered(responseCode int, now time.Time) {
	wl.Status = WebhookStatusDelivered
	wl.ResponseCode = &responseCode
	wl.ErrorMessage = nil
	wl.DeliveredAt = &now
	wl.UpdatedAt = now
}
