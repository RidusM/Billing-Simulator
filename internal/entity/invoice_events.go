package entity

import (
	"time"

	"github.com/google/uuid"
)

// InvoiceCreatedEvent
type InvoiceCreatedEvent struct {
	InvoiceID         uuid.UUID
	InvoicePubID      string
	CustomerID        uuid.UUID
	CustomerPubID     string
	SubscriptionID    *uuid.UUID
	SubscriptionPubID *string
	Amount            int64
	Currency          string
	Status            InvoiceStatus
	CreatedAt         time.Time
}

func (e InvoiceCreatedEvent) EventType() string      { return "invoice.created" }
func (e InvoiceCreatedEvent) OccurredOn() time.Time  { return e.CreatedAt }
func (e InvoiceCreatedEvent) AggregateID() uuid.UUID { return e.InvoiceID }
func (e InvoiceCreatedEvent) AggregateType() string  { return "invoice" }

// InvoicePaidEvent
type InvoicePaidEvent struct {
	InvoiceID      uuid.UUID
	InvoicePubID   string
	CustomerID     uuid.UUID
	CustomerPubID  string
	SubscriptionID *uuid.UUID
	Amount         int64
	Currency       string
	PaidAt         time.Time
}

func (e InvoicePaidEvent) EventType() string      { return "invoice.paid" }
func (e InvoicePaidEvent) OccurredOn() time.Time  { return e.PaidAt }
func (e InvoicePaidEvent) AggregateID() uuid.UUID { return e.InvoiceID }
func (e InvoicePaidEvent) AggregateType() string  { return "invoice" }

// InvoicePaymentFailedEvent
type InvoicePaymentFailedEvent struct {
	InvoiceID      uuid.UUID
	InvoicePubID   string
	CustomerID     uuid.UUID
	CustomerPubID  string
	SubscriptionID *uuid.UUID
	Amount         int64
	Currency       string
	ErrorCode      string
	FailedAt       time.Time
}

func (e InvoicePaymentFailedEvent) EventType() string      { return "invoice.payment_failed" }
func (e InvoicePaymentFailedEvent) OccurredOn() time.Time  { return e.FailedAt }
func (e InvoicePaymentFailedEvent) AggregateID() uuid.UUID { return e.InvoiceID }
func (e InvoicePaymentFailedEvent) AggregateType() string  { return "invoice" }
