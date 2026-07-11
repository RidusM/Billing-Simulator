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
	utc := now.UTC()
	return &OutboxEvent{
		ID:            uuid.New(),
		EventType:     string(event.EventType()),
		AggregateID:   event.AggregateID(),
		AggregateType: event.AggregateType(),
		Payload:       payload,
		OccurredAt:    event.OccurredOn(),
		Processed:     false,
		CreatedAt:     utc,
	}
}

func (e *OutboxEvent) MarkProcessed(now time.Time) {
	utc := now.UTC()
	e.Processed = true
	e.ProcessedAt = &utc
}

func (o *OutboxEvent) MarkFailed(
	err string,
	nextRetry time.Time,
) {

	o.Attempt++

	o.NextAttemptAt = &nextRetry

	o.Error = &err
}
