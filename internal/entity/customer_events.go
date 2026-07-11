package entity

import (
	"time"

	"github.com/google/uuid"
)

// CustomerCreatedEvent
type CustomerCreatedEvent struct {
	CustomerID    uuid.UUID
	CustomerPubID string
	Email         string
	Name          string
	CreatedAt     time.Time
}

func (e CustomerCreatedEvent) EventType() EventType   { return EventCustomerCreated }
func (e CustomerCreatedEvent) OccurredOn() time.Time  { return e.CreatedAt }
func (e CustomerCreatedEvent) AggregateID() uuid.UUID { return e.CustomerID }
func (e CustomerCreatedEvent) AggregateType() string  { return "customer" }

type CustomerUpdatedEvent struct {
	CustomerID    uuid.UUID
	CustomerPubID string
	Email         string
	Name          string
	UpdatedAt     time.Time
}

func (e CustomerUpdatedEvent) EventType() EventType   { return EventCustomerUpdated }
func (e CustomerUpdatedEvent) OccurredOn() time.Time  { return e.UpdatedAt }
func (e CustomerUpdatedEvent) AggregateID() uuid.UUID { return e.CustomerID }
func (e CustomerUpdatedEvent) AggregateType() string  { return "customer" }
