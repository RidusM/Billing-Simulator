package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventCustomerCreated        EventType = "customer.created"
	EventCustomerUpdated        EventType = "customer.updated"
	EventPriceCreated           EventType = "price.created"
	EventPriceUpdated           EventType = "price.updated"
	EventProductCreated         EventType = "product.created"
	EventProductUpdated         EventType = "product.updated"
	EventSubscriptionCreated    EventType = "subscription.created"
	EventSubscriptionRenewed    EventType = "subscription.renewed"
	EventSubscriptionUpdated    EventType = "subscription.updated"
	EventSubscriptionCanceled   EventType = "subscription.canceled"
	EventInvoiceCreated         EventType = "invoice.created"
	EventInvoicePaid            EventType = "invoice.paid"
	EventInvoicePaymentFailed   EventType = "invoice.payment_failed"
	EventPaymentIntentFailed    EventType = "payment_intent.payment_failed"
	EventPaymentIntentSucceeded EventType = "payment_intent.succeeded"
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
	pubID, err := GeneratePublicID("evt")
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	utc := now.UTC()
	return &Event{
		ID:         uuid.New(),
		PublicID:   pubID,
		Type:       eventType,
		APIVersion: "2024-01-01",
		Payload:    data,
		CreatedAt:  utc,
	}, nil
}
