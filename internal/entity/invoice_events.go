package entity

import (
	"time"

	"github.com/google/uuid"
)

// InvoiceCreatedEvent
type InvoiceCreatedEvent struct {
	InvoiceID         uuid.UUID     `json:"invoice_id"`
	InvoicePubID      string        `json:"invoice_public_id"`
	CustomerID        uuid.UUID     `json:"customer_id"`
	CustomerPubID     string        `json:"customer_public_id"`
	SubscriptionID    *uuid.UUID    `json:"subscription_id,omitempty"`
	SubscriptionPubID *string       `json:"subscription_public_id,omitempty"`
	Amount            int64         `json:"amount"`
	Currency          string        `json:"currency"`
	Status            InvoiceStatus `json:"status"`
	CreatedAt         time.Time     `json:"created_at"`
}

func (e InvoiceCreatedEvent) EventType() string      { return "invoice.created" }
func (e InvoiceCreatedEvent) OccurredOn() time.Time  { return e.CreatedAt }
func (e InvoiceCreatedEvent) AggregateID() uuid.UUID { return e.InvoiceID }

// InvoicePaidEvent
type InvoicePaidEvent struct {
	InvoiceID      uuid.UUID  `json:"invoice_id"`
	InvoicePubID   string     `json:"invoice_public_id"`
	CustomerID     uuid.UUID  `json:"customer_id"`
	CustomerPubID  string     `json:"customer_public_id"`
	SubscriptionID *uuid.UUID `json:"subscription_id,omitempty"`
	Amount         int64      `json:"amount"`
	Currency       string     `json:"currency"`
	PaidAt         time.Time  `json:"paid_at"`
}

func (e InvoicePaidEvent) EventType() string      { return "invoice.paid" }
func (e InvoicePaidEvent) OccurredOn() time.Time  { return e.PaidAt }
func (e InvoicePaidEvent) AggregateID() uuid.UUID { return e.InvoiceID }

// InvoicePaymentFailedEvent
type InvoicePaymentFailedEvent struct {
	InvoiceID      uuid.UUID  `json:"invoice_id"`
	InvoicePubID   string     `json:"invoice_public_id"`
	CustomerID     uuid.UUID  `json:"customer_id"`
	CustomerPubID  string     `json:"customer_public_id"`
	SubscriptionID *uuid.UUID `json:"subscription_id,omitempty"`
	Amount         int64      `json:"amount"`
	Currency       string     `json:"currency"`
	ErrorCode      string     `json:"error_code"`
	FailedAt       time.Time  `json:"failed_at"`
}

func (e InvoicePaymentFailedEvent) EventType() string      { return "invoice.payment_failed" }
func (e InvoicePaymentFailedEvent) OccurredOn() time.Time  { return e.FailedAt }
func (e InvoicePaymentFailedEvent) AggregateID() uuid.UUID { return e.InvoiceID }
