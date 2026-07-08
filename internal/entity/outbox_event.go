package entity

import (
	"time"

	"github.com/google/uuid"
)

type OutboxEvent struct {
	ID            uuid.UUID
	EventType     string
	AggregateID   uuid.UUID
	AggregateType string
	Payload       []byte
	OccurredAt    time.Time
	Processed     bool
	ProcessedAt   *time.Time
	Attempt       int
	NextAttemptAt *time.Time
	Error         *string
	CreatedAt     time.Time
}

func NewOutboxEvent(event DomainEvent, now time.Time, payload []byte) *OutboxEvent {
	return &OutboxEvent{
		ID:            uuid.New(),
		EventType:     event.EventType(),
		AggregateID:   event.AggregateID(),
		AggregateType: event.AggregateType(),
		Payload:       payload,
		OccurredAt:    event.OccurredOn(),
		Processed:     false,
		CreatedAt:     now,
	}
}

func (e *OutboxEvent) MarkProcessed(now time.Time) {
	e.Processed = true
	e.ProcessedAt = &now
}

func (e *OutboxEvent) MarkFailed(errMsg string) {
	e.Error = &errMsg
}
