package entity

import (
	"time"

	"github.com/google/uuid"
)

type OutboxEvent struct {
	ID          uuid.UUID
	EventType   string
	AggregateID uuid.UUID
	Payload     []byte
	OccurredAt  time.Time
	Processed   bool
	ProcessedAt *time.Time
	Error       *string
	CreatedAt   time.Time
}

func NewOutboxEvent(event DomainEvent, payload []byte) *OutboxEvent {
	now := time.Now().UTC()
	return &OutboxEvent{
		ID:          uuid.New(),
		EventType:   event.EventType(),
		AggregateID: event.AggregateID(),
		Payload:     payload,
		OccurredAt:  event.OccurredOn(),
		Processed:   false,
		CreatedAt:   now,
	}
}

func (e *OutboxEvent) MarkProcessed() {
	now := time.Now().UTC()
	e.Processed = true
	e.ProcessedAt = &now
}

func (e *OutboxEvent) MarkFailed(errMsg string) {
	e.Error = &errMsg
}
