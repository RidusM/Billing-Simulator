package entity

import (
	"time"

	"github.com/google/uuid"
)

type PaymentIntentSucceededEvent struct {
	PaymentIntentID    uuid.UUID
	PaymentIntentPubID string
	CustomerID         uuid.UUID
	InvoiceID          *uuid.UUID
	Amount             int64
	Currency           string
	SucceededAt        time.Time
}

func (e PaymentIntentSucceededEvent) EventType() EventType   { return EventPaymentIntentSucceeded }
func (e PaymentIntentSucceededEvent) OccurredOn() time.Time  { return e.SucceededAt }
func (e PaymentIntentSucceededEvent) AggregateID() uuid.UUID { return e.PaymentIntentID }
func (e PaymentIntentSucceededEvent) AggregateType() string  { return "payment_intent" }

type PaymentIntentFailedEvent struct {
	PaymentIntentID    uuid.UUID
	PaymentIntentPubID string
	CustomerID         uuid.UUID
	InvoiceID          *uuid.UUID
	Amount             int64
	Currency           string
	ErrorCode          string
	DeclineCode        string
	FailedAt           time.Time
}

func (e PaymentIntentFailedEvent) EventType() EventType   { return EventPaymentIntentFailed }
func (e PaymentIntentFailedEvent) OccurredOn() time.Time  { return e.FailedAt }
func (e PaymentIntentFailedEvent) AggregateID() uuid.UUID { return e.PaymentIntentID }
func (e PaymentIntentFailedEvent) AggregateType() string  { return "payment_intent" }
