package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type EventType string

// TODO: ИЗМЕНИТЬ!!!!!!!!!
const (
	EventCustomerCreated      EventType = "customer.created"
	EventSubscriptionCreated  EventType = "subscription.created"
	EventSubscriptionRenewed  EventType = "subscription.renewed"
	EventSubscriptionCanceled EventType = "subscription.canceled"
	EventInvoiceCreated       EventType = "invoice.created"
	EventInvoicePaid          EventType = "invoice.paid"
	EventInvoiceFailed        EventType = "invoice.payment_failed"
	EventPaymentIntentFailed  EventType = "payment_intent.payment_failed"
)

type Event struct {
	ID             uuid.UUID
	PublicID       string
	Type           EventType
	APIVersion     string
	Payload        json.RawMessage
	IdempotencyKey *string
	CreatedAt      time.Time
}

func NewEvent(eventType EventType, payload any, now time.Time) (*Event, error) {
	pubID, _ := GeneratePublicID("evt")

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &Event{
		ID:         uuid.New(),
		PublicID:   pubID,
		Type:       eventType,
		APIVersion: "2024-01-01",
		Payload:    data,
		CreatedAt:  now.UTC(),
	}, nil
}
