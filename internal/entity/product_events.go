package entity

import (
	"time"

	"github.com/google/uuid"
)

type ProductCreatedEvent struct {
	ProductID    uuid.UUID
	ProductPubID string
	Name         string
	Description  string
	CreatedAt    time.Time
}

func (e ProductCreatedEvent) EventType() EventType   { return EventProductCreated }
func (e ProductCreatedEvent) OccurredOn() time.Time  { return e.CreatedAt }
func (e ProductCreatedEvent) AggregateID() uuid.UUID { return e.ProductID }
func (e ProductCreatedEvent) AggregateType() string  { return "product" }

type ProductUpdatedEvent struct {
	ProductID    uuid.UUID
	ProductPubID string
	Name         string
	Description  string
	Active       bool
	UpdatedAt    time.Time
}

func (e ProductUpdatedEvent) EventType() EventType   { return EventProductUpdated }
func (e ProductUpdatedEvent) OccurredOn() time.Time  { return e.UpdatedAt }
func (e ProductUpdatedEvent) AggregateID() uuid.UUID { return e.ProductID }
func (e ProductUpdatedEvent) AggregateType() string  { return "product" }
