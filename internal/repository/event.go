package repository

import (
	"context"
	"fmt"

	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/storage/postgres"
	"bill-stripe-sim/pkg/storage/postgres/transaction"

	"github.com/Masterminds/squirrel"
)

type EventRepository struct {
	storage *postgres.Postgres
}

func NewEventRepository(storage *postgres.Postgres) *EventRepository {
	return &EventRepository{
		storage: storage,
	}
}

func (r *EventRepository) Create(ctx context.Context, e *entity.Event) error {
	const op = "repository.event.Create"
	sql, args, err := r.storage.Builder.
		Insert("events").
		Columns("id", "event_type", "payload", "created_at").
		Values(e.ID, e.Type, e.Payload, e.CreatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = r.executor(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *EventRepository) GetByType(ctx context.Context, eventType entity.EventType, limit int) ([]*entity.Event, error) {
	const op = "repository.event.GetByType"
	sql, args, err := r.storage.
		Select("id", "event_type", "payload", "created_at").
		From("events").
		Where(squirrel.Eq{"event_type": eventType}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := r.executor(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var events []*entity.Event
	for rows.Next() {
		var e entity.Event
		err = rows.Scan(
			&e.ID,
			&e.Type,
			&e.Payload,
			&e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		events = append(events, &e)
	}
	return events, nil
}

func (r *EventRepository) GetRecent(ctx context.Context, limit int) ([]*entity.Event, error) {
	const op = "repository.event.GetRecent"
	sql, args, err := r.storage.
		Select("id", "event_type", "payload", "created_at").
		From("events").
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := r.executor(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var events []*entity.Event
	for rows.Next() {
		var e entity.Event
		err = rows.Scan(
			&e.ID,
			&e.Type,
			&e.Payload,
			&e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		events = append(events, &e)
	}
	return events, nil
}

func (r *EventRepository) executor(ctx context.Context) postgres.QueryExecuter {
	if qe, ok := transaction.TxFromCtx(ctx); ok {
		return qe
	}
	return r.storage
}
