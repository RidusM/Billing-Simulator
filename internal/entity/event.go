package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventCustomerCreated      EventType = "customer.created"
	EventSubscriptionCreated  EventType = "subscription.created"
	EventSubscriptionRenewed  EventType = "subscription.renewed"
	EventSubscriptionCanceled EventType = "subscription.canceled"
	EventInvoiceCreated       EventType = "invoice.created"
	EventInvoicePaid          EventType = "invoice.paid"
	EventInvoiceFailed        EventType = "invoice.payment_failed"
	EventWebhookDelivered     EventType = "webhook.delivered"
	EventWebhookFailed        EventType = "webhook.failed"
)

type Event struct {
	ID        uuid.UUID
	Type      EventType
	Payload   json.RawMessage
	CreatedAt time.Time
}

func NewEvent(eventType EventType, payload any, now time.Time) (*Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &Event{
		ID:        uuid.New(),
		Type:      eventType,
		Payload:   data,
		CreatedAt: now.UTC(),
	}, nil
}
