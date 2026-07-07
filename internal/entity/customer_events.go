package entity

import (
	"time"

	"github.com/google/uuid"
)

// CustomerCreatedEvent
type CustomerCreatedEvent struct {
	CustomerID    uuid.UUID `json:"customer_id"`
	CustomerPubID string    `json:"customer_public_id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	CreatedAt     time.Time `json:"created_at"`
}

func (e CustomerCreatedEvent) EventType() string      { return "customer.created" }
func (e CustomerCreatedEvent) OccurredOn() time.Time  { return e.CreatedAt }
func (e CustomerCreatedEvent) AggregateID() uuid.UUID { return e.CustomerID }
