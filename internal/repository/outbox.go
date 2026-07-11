package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/storage/postgres"
	"bill-stripe-sim/pkg/storage/postgres/transaction"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type OutboxRepository struct {
	storage *postgres.Postgres
}

func NewOutboxRepository(storage *postgres.Postgres) *OutboxRepository {
	return &OutboxRepository{
		storage: storage,
	}
}

// Save — сохраняет одно событие в outbox со всеми новыми полями
func (r *OutboxRepository) Save(ctx context.Context, event *entity.OutboxEvent) error {
	const op = "repository.outbox.Save"

	sql, args, err := r.storage.Builder.
		Insert("outbox_events").
		Columns(
			"id", "event_type", "aggregate_id", "aggregate_type",
			"payload", "occurred_at", "created_at", "processed",
			"attempt", "next_attempt_at",
		).
		Values(
			event.ID,
			event.EventType,
			event.AggregateID,
			event.AggregateType, // ← Добавлено
			event.Payload,
			event.OccurredAt,
			event.CreatedAt,
			event.Processed,
			event.Attempt,       // ← Добавлено
			event.NextAttemptAt, // ← Добавлено
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = r.executor(ctx).Exec(ctx, sql, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%s: %w", op, entity.ErrConflictingData)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// SaveBatch — оптимизированный массовый INSERT
func (r *OutboxRepository) SaveBatch(ctx context.Context, events []*entity.OutboxEvent) error {
	const op = "repository.outbox.SaveBatch"

	if len(events) == 0 {
		return nil
	}

	builder := r.storage.Builder.Insert("outbox_events").
		Columns(
			"id", "event_type", "aggregate_id", "aggregate_type",
			"payload", "occurred_at", "created_at", "processed",
			"attempt", "next_attempt_at",
		)

	for _, event := range events {
		builder = builder.Values(
			event.ID,
			event.EventType,
			event.AggregateID,
			event.AggregateType, // ← Добавлено
			event.Payload,
			event.OccurredAt,
			event.CreatedAt,
			event.Processed,
			event.Attempt,       // ← Добавлено
			event.NextAttemptAt, // ← Добавлено
		)
	}

	sql, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = r.executor(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *OutboxRepository) GetUnprocessed(ctx context.Context, limit int, olderThan time.Duration) ([]*entity.OutboxEvent, error) {
	const op = "repository.outbox.GetUnprocessed"
	sql, args, err := r.storage.
		Select(
			"id", "event_type", "aggregate_id", "aggregate_type",
			"payload", "occurred_at", "created_at", "processed",
			"processed_at", "error", "attempt", "next_attempt_at",
		).
		From("outbox_events").
		Where(squirrel.And{
			squirrel.Eq{"processed": false},
			squirrel.LtOrEq{"attempt": 5},
			squirrel.Or{
				squirrel.Expr("next_attempt_at IS NULL"),
				squirrel.LtOrEq{"next_attempt_at": time.Now().UTC()},
			},
		}).
		OrderBy("next_attempt_at ASC NULLS FIRST, created_at ASC").
		Limit(uint64(limit)).
		Suffix("FOR UPDATE SKIP LOCKED").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := r.executor(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var events []*entity.OutboxEvent
	for rows.Next() {
		var e entity.OutboxEvent
		err := rows.Scan(
			&e.ID, &e.EventType, &e.AggregateID, &e.AggregateType, // ← Читаем AggregateType
			&e.Payload, &e.OccurredAt, &e.CreatedAt, &e.Processed,
			&e.ProcessedAt, &e.Error, &e.Attempt, &e.NextAttemptAt, // ← Читаем Attempt и NextAttemptAt
		)
		if err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		events = append(events, &e)
	}
	return events, nil
}

func (r *OutboxRepository) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	const op = "repository.outbox.MarkProcessed"

	sql, args, err := r.storage.Builder.
		Update("outbox_events").
		Set("processed", true).
		Set("processed_at", squirrel.Expr("NOW()")).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	tag, err := r.executor(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, entity.ErrDataNotFound)
	}

	return nil
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, id uuid.UUID, errorMsg string, attempt int) error {
	const op = "repository.outbox.MarkFailed"

	backoffSeconds := int64(math.Pow(2, float64(attempt)))
	nextAttemptAt := time.Now().UTC().Add(time.Duration(backoffSeconds) * time.Second)

	sql, args, err := r.storage.Builder.
		Update("outbox_events").
		Set("error", errorMsg).
		Set("attempt", attempt).
		Set("next_attempt_at", nextAttemptAt).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	tag, err := r.executor(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, entity.ErrDataNotFound)
	}

	return nil
}

func (r *OutboxRepository) DeleteOldProcessed(ctx context.Context, olderThan time.Duration) (int64, error) {
	const op = "repository.outbox.DeleteOldProcessed"

	sql, args, err := r.storage.Builder.
		Delete("outbox_events").
		Where(squirrel.And{
			squirrel.Eq{"processed": true},
			squirrel.Expr("processed_at <= NOW() - ? * INTERVAL '1 SECOND'", olderThan.Seconds()),
		}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	tag, err := r.executor(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return tag.RowsAffected(), nil
}

// scanOutboxEvents — хелпер, который тоже приводим к соответствию новой схеме
func scanOutboxEvents(op string, rows pgx.Rows) ([]*entity.OutboxEvent, error) {
	var events []*entity.OutboxEvent

	for rows.Next() {
		var e entity.OutboxEvent
		err := rows.Scan(
			&e.ID,
			&e.EventType,
			&e.AggregateID,
			&e.AggregateType, // ← Добавлено
			&e.Payload,
			&e.OccurredAt,
			&e.CreatedAt,
			&e.Processed,
			&e.ProcessedAt,
			&e.Error,
			&e.Attempt,       // ← Добавлено
			&e.NextAttemptAt, // ← Добавлено
		)
		if err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		events = append(events, &e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return events, nil
}

func (r *OutboxRepository) executor(ctx context.Context) postgres.QueryExecuter {
	if qe, ok := transaction.TxFromCtx(ctx); ok {
		return qe
	}
	return r.storage
}
