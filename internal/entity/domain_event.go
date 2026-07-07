package entity

import (
	"time"

	"github.com/google/uuid"
)

type DomainEvent interface {
	EventType() string
	OccurredOn() time.Time
	AggregateID() uuid.UUID
}

type DomainEvents []DomainEvent

func (de *DomainEvents) AppendIfUnique(event DomainEvent) {
	eventType := event.EventType()
	for _, e := range *de {
		if e.EventType() == eventType {
			return
		}
	}
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
