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

func NewWebhookLog(eventID, endpointID, traceID uuid.UUID, eventType string, payload json.RawMessage, targetURL string, now time.Time) (*WebhookLog, error) {
	pubID, err := GeneratePublicID("wh")
	if err != nil {
		return nil, err
	}

	utc := now.UTC()

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
		NextAttemptAt: utc,
		CreatedAt:     utc,
		UpdatedAt:     utc,
	}, nil
}

func (wl *WebhookLog) MarkDelivered(responseCode int, now time.Time) {
	utc := now.UTC()

	wl.Status = WebhookStatusDelivered
	wl.ResponseCode = &responseCode
	wl.ErrorMessage = nil
	wl.DeliveredAt = &utc
	wl.UpdatedAt = utc
}

func (wl *WebhookLog) MarkFailed(errMsg string, nextAttempt time.Time) {
	now := time.Now().UTC()
	wl.ErrorMessage = &errMsg
	wl.Attempt++
	wl.NextAttemptAt = nextAttempt.UTC()
	wl.UpdatedAt = now

	if wl.Attempt >= wl.MaxAttempts {
		wl.Status = WebhookStatusFailed
	} else {
		wl.Status = WebhookStatusPending
	}
}
