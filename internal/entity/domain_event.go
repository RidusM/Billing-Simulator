package entity

import (
	"time"

	"github.com/google/uuid"
)

type DomainEvent interface {
	EventType() string
	OccurredOn() time.Time
	AggregateID() uuid.UUID
	AggregateType() string // ← Добавлено
}

type DomainEvents []DomainEvent

func (de *DomainEvents) ClearAndReturn() DomainEvents {
	events := *de
	*de = nil
	return events
}

func (de *DomainEvents) HasEvents() bool {
	return len(*de) > 0
}
