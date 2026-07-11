package entity

import (
	"time"

	"github.com/google/uuid"
)

type SubscriptionCreatedEvent struct {
	SubscriptionID    uuid.UUID
	SubscriptionPubID string
	CustomerID        uuid.UUID
	CustomerPubID     string
	PriceID           uuid.UUID
	PricePubID        string
	Status            SubscriptionStatus
	CurrentPeriodEnd  time.Time
	NextBillingAt     time.Time
	CreatedAt         time.Time
}

func (e SubscriptionCreatedEvent) EventType() EventType   { return EventSubscriptionCreated }
func (e SubscriptionCreatedEvent) OccurredOn() time.Time  { return e.CreatedAt }
func (e SubscriptionCreatedEvent) AggregateID() uuid.UUID { return e.SubscriptionID }
func (e SubscriptionCreatedEvent) AggregateType() string  { return "subscription" }

// SubscriptionCanceledEvent
type SubscriptionCanceledEvent struct {
	SubscriptionID    uuid.UUID
	SubscriptionPubID string
	CustomerID        uuid.UUID
	CustomerPubID     string
	Status            SubscriptionStatus
	CanceledAt        time.Time
	AtPeriodEnd       bool
}

func (e SubscriptionCanceledEvent) EventType() EventType   { return EventSubscriptionCanceled }
func (e SubscriptionCanceledEvent) OccurredOn() time.Time  { return e.CanceledAt }
func (e SubscriptionCanceledEvent) AggregateID() uuid.UUID { return e.SubscriptionID }
func (e SubscriptionCanceledEvent) AggregateType() string  { return "subscription" }

// SubscriptionRenewedEvent
type SubscriptionRenewedEvent struct {
	SubscriptionID    uuid.UUID
	SubscriptionPubID string
	CustomerID        uuid.UUID
	CustomerPubID     string
	InvoiceID         uuid.UUID
	InvoicePubID      string
	InvoiceAmount     int64
	InvoiceCurrency   string
	InvoiceStatus     InvoiceStatus
	NewPeriodEnd      time.Time
	RenewedAt         time.Time
}

func (e SubscriptionRenewedEvent) EventType() EventType   { return EventSubscriptionRenewed }
func (e SubscriptionRenewedEvent) OccurredOn() time.Time  { return e.RenewedAt }
func (e SubscriptionRenewedEvent) AggregateID() uuid.UUID { return e.SubscriptionID }
func (e SubscriptionRenewedEvent) AggregateType() string  { return "subscription" }

type SubscriptionUpdatedEvent struct {
	SubscriptionID    uuid.UUID
	SubscriptionPubID string
	CustomerID        uuid.UUID
	CustomerPubID     string
	Status            SubscriptionStatus
	CancelAtPeriodEnd bool
	UpdatedAt         time.Time
}

func (e SubscriptionUpdatedEvent) EventType() EventType   { return EventSubscriptionUpdated } // Строго по стандарту Stripe!
func (e SubscriptionUpdatedEvent) OccurredOn() time.Time  { return e.UpdatedAt }
func (e SubscriptionUpdatedEvent) AggregateID() uuid.UUID { return e.SubscriptionID }
func (e SubscriptionUpdatedEvent) AggregateType() string  { return "subscription" }
