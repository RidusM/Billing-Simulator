package entity

import (
	"time"

	"github.com/google/uuid"
)

type SubscriptionCreatedEvent struct {
	SubscriptionID    uuid.UUID          `json:"subscription_id"`
	SubscriptionPubID string             `json:"subscription_public_id"`
	CustomerID        uuid.UUID          `json:"customer_id"`
	CustomerPubID     string             `json:"customer_public_id"`
	PriceID           uuid.UUID          `json:"price_id"`
	PricePubID        string             `json:"price_public_id"`
	Status            SubscriptionStatus `json:"status"`
	CurrentPeriodEnd  time.Time          `json:"current_period_end"`
	NextBillingAt     time.Time          `json:"next_billing_at"`
	CreatedAt         time.Time          `json:"created_at"`
}

func (e SubscriptionCreatedEvent) EventType() string      { return "subscription.created" }
func (e SubscriptionCreatedEvent) OccurredOn() time.Time  { return e.CreatedAt }
func (e SubscriptionCreatedEvent) AggregateID() uuid.UUID { return e.SubscriptionID }
func (e SubscriptionCreatedEvent) AggregateType() string  { return "subscription" }

// SubscriptionCanceledEvent
type SubscriptionCanceledEvent struct {
	SubscriptionID    uuid.UUID          `json:"subscription_id"`
	SubscriptionPubID string             `json:"subscription_public_id"`
	CustomerID        uuid.UUID          `json:"customer_id"`
	CustomerPubID     string             `json:"customer_public_id"`
	Status            SubscriptionStatus `json:"status"`
	CanceledAt        time.Time          `json:"canceled_at"`
	AtPeriodEnd       bool               `json:"at_period_end"`
}

func (e SubscriptionCanceledEvent) EventType() string      { return "subscription.canceled" }
func (e SubscriptionCanceledEvent) OccurredOn() time.Time  { return e.CanceledAt }
func (e SubscriptionCanceledEvent) AggregateID() uuid.UUID { return e.SubscriptionID }
func (e SubscriptionCanceledEvent) AggregateType() string  { return "subscription" }

// SubscriptionRenewedEvent
type SubscriptionRenewedEvent struct {
	SubscriptionID    uuid.UUID     `json:"subscription_id"`
	SubscriptionPubID string        `json:"subscription_public_id"`
	CustomerID        uuid.UUID     `json:"customer_id"`
	CustomerPubID     string        `json:"customer_public_id"`
	InvoiceID         uuid.UUID     `json:"invoice_id"`
	InvoicePubID      string        `json:"invoice_public_id"`
	InvoiceAmount     int64         `json:"invoice_amount"`
	InvoiceCurrency   string        `json:"invoice_currency"`
	InvoiceStatus     InvoiceStatus `json:"invoice_status"`
	NewPeriodEnd      time.Time     `json:"new_period_end"`
	RenewedAt         time.Time     `json:"renewed_at"`
}

func (e SubscriptionRenewedEvent) EventType() string      { return "subscription.renewed" }
func (e SubscriptionRenewedEvent) OccurredOn() time.Time  { return e.RenewedAt }
func (e SubscriptionRenewedEvent) AggregateID() uuid.UUID { return e.SubscriptionID }
func (e SubscriptionRenewedEvent) AggregateType() string  { return "subscription" }
