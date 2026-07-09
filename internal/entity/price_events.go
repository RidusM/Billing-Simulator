package entity

import (
	"time"

	"github.com/google/uuid"
)

type PriceCreatedEvent struct {
	PriceID       uuid.UUID
	PricePubID    string
	ProductID     uuid.UUID
	Amount        int64
	Currency      string
	Interval      BillingInterval
	IntervalCount int
	CreatedAt     time.Time
}

func (e PriceCreatedEvent) EventType() string      { return "price.created" }
func (e PriceCreatedEvent) OccurredOn() time.Time  { return e.CreatedAt }
func (e PriceCreatedEvent) AggregateID() uuid.UUID { return e.PriceID }
func (e PriceCreatedEvent) AggregateType() string  { return "price" }

type PriceUpdatedEvent struct {
	PriceID       uuid.UUID
	PricePubID    string
	ProductID     uuid.UUID
	Amount        int64
	Currency      string
	Interval      BillingInterval
	IntervalCount int
	Active        bool
	UpdatedAt     time.Time
}

func (e PriceUpdatedEvent) EventType() string      { return "price.updated" }
func (e PriceUpdatedEvent) OccurredOn() time.Time  { return e.UpdatedAt }
func (e PriceUpdatedEvent) AggregateID() uuid.UUID { return e.PriceID }
func (e PriceUpdatedEvent) AggregateType() string  { return "price" }
