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

func (e CustomerCreatedEvent) EventType() string      { return "customer.created" }
func (e CustomerCreatedEvent) OccurredOn() time.Time  { return e.CreatedAt }
func (e CustomerCreatedEvent) AggregateID() uuid.UUID { return e.CustomerID }
func (e CustomerCreatedEvent) AggregateType() string  { return "customer" }
