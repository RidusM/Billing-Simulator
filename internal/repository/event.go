package repository

import (
	"context"
	"errors"
	"fmt"

	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/storage/postgres"
	"bill-stripe-sim/pkg/storage/postgres/transaction"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
		Columns("id", "public_id", "event_type", "api_version", "payload", "idempotency_key", "created_at").
		Values(e.ID, e.PublicID, e.Type, e.APIVersion, e.Payload, e.IdempotencyKey, e.CreatedAt).
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

func (r *EventRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
	const op = "repository.event.GetByID"
	sql, args, err := r.storage.
		Select("id", "public_id", "event_type", "api_version", "payload", "idempotency_key", "created_at").
		From("events").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var e entity.Event
	err = r.executor(ctx).QueryRow(ctx, sql, args...).Scan(
		&e.ID,
		&e.PublicID,
		&e.Type,
		&e.APIVersion,
		&e.Payload,
		&e.IdempotencyKey,
		&e.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, entity.ErrDataNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &e, nil
}

func (r *EventRepository) GetByType(ctx context.Context, eventType entity.EventType, limit, offset int) ([]*entity.Event, error) {
	const op = "repository.event.GetByType"
	sql, args, err := r.storage.
		Select("id", "public_id", "event_type", "api_version", "payload", "idempotency_key", "created_at").
		From("events").
		Where(squirrel.Eq{"event_type": eventType}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
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
			&e.PublicID,
			&e.Type,
			&e.APIVersion,
			&e.Payload,
			&e.IdempotencyKey,
			&e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		events = append(events, &e)
	}
	return events, nil
}

func (r *EventRepository) GetRecent(ctx context.Context, limit, offset int) ([]*entity.Event, error) {
	const op = "repository.event.GetRecent"
	sql, args, err := r.storage.
		Select("id", "public_id", "event_type", "api_version", "payload", "idempotency_key", "created_at").
		From("events").
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
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
			&e.PublicID,
			&e.Type,
			&e.APIVersion,
			&e.Payload,
			&e.IdempotencyKey,
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
