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
	ID           uuid.UUID
	TraceID      uuid.UUID
	EventType    string
	Payload      json.RawMessage
	TargetURL    string
	Status       WebhookStatus
	ResponseCode *int
	Attempt      int
	ErrorMessage *string
	CreatedAt    time.Time
}
