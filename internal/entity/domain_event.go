package entity

import (
	"time"

	"github.com/google/uuid"
)

type DomainEvent interface {
	EventType() EventType
	OccurredOn() time.Time
	AggregateID() uuid.UUID
	AggregateType() string
}

type DomainEvents []DomainEvent

func (de *DomainEvents) Raise(event DomainEvent) {
	*de = append(*de, event)
}

func (de *DomainEvents) ClearAndReturn() DomainEvents {
	events := *de
	*de = nil
	return events
}

func (de *DomainEvents) HasEvents() bool {
	return len(*de) > 0
}
