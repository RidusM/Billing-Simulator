package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/storage/postgres"
	"bill-stripe-sim/pkg/storage/postgres/transaction"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// OutboxRepository — реализация репозитория для Transactional Outbox
type OutboxRepository struct {
	storage *postgres.Postgres
}

func NewOutboxRepository(storage *postgres.Postgres) *OutboxRepository {
	return &OutboxRepository{
		storage: storage,
	}
}

// Save — сохраняет одно событие в outbox
func (r *OutboxRepository) Save(ctx context.Context, event *entity.OutboxEvent) error {
	const op = "repository.outbox.Save"

	sql, args, err := r.storage.Builder.
		Insert("outbox_events").
		Columns("id", "event_type", "aggregate_id", "payload", "occurred_at", "created_at", "processed").
		Values(
			event.ID,
			event.EventType,
			event.AggregateID,
			event.Payload,
			event.OccurredAt,
			event.CreatedAt,
			event.Processed,
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

// SaveBatch — сохраняет несколько событий в outbox (оптимизировано через один INSERT)
func (r *OutboxRepository) SaveBatch(ctx context.Context, events []*entity.OutboxEvent) error {
	const op = "repository.outbox.SaveBatch"

	if len(events) == 0 {
		return nil
	}

	// Строим массовый INSERT
	builder := r.storage.Builder.Insert("outbox_events").
		Columns("id", "event_type", "aggregate_id", "payload", "occurred_at", "created_at", "processed")

	for _, event := range events {
		builder = builder.Values(
			event.ID,
			event.EventType,
			event.AggregateID,
			event.Payload,
			event.OccurredAt,
			event.CreatedAt,
			event.Processed,
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

// GetUnprocessed — получает необработанные события для обработки воркером
// olderThan — минимальный возраст события (защита от чтения незакоммиченных транзакций)
func (r *OutboxRepository) GetUnprocessed(ctx context.Context, limit int, olderThan time.Duration) ([]*entity.OutboxEvent, error) {
	const op = "repository.outbox.GetUnprocessed"

	// Вычисляем пороговое время
	cutoffTime := time.Now().UTC().Add(-olderThan)

	sql, args, err := r.storage.
		Select("id", "event_type", "aggregate_id", "payload", "occurred_at", "created_at", "processed", "processed_at", "error").
		From("outbox_events").
		Where(squirrel.And{
			squirrel.Eq{"processed": false},
			squirrel.LtOrEq{"created_at": cutoffTime},
		}).
		OrderBy("created_at ASC").
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

	return scanOutboxEvents(op, rows)
}

// MarkProcessed — помечает событие как успешно обработанное
func (r *OutboxRepository) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	const op = "repository.outbox.MarkProcessed"

	now := time.Now().UTC()
	sql, args, err := r.storage.Builder.
		Update("outbox_events").
		Set("processed", true).
		Set("processed_at", now).
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

// MarkFailed — помечает событие как ошибочное (для retry или DLQ)
func (r *OutboxRepository) MarkFailed(ctx context.Context, id uuid.UUID, errorMsg string) error {
	const op = "repository.outbox.MarkFailed"

	sql, args, err := r.storage.Builder.
		Update("outbox_events").
		Set("error", errorMsg).
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

// DeleteOldProcessed — удаляет старые обработанные события (cleanup)
func (r *OutboxRepository) DeleteOldProcessed(ctx context.Context, olderThan time.Duration) (int64, error) {
	const op = "repository.outbox.DeleteOldProcessed"

	cutoffTime := time.Now().UTC().Add(-olderThan)

	sql, args, err := r.storage.Builder.
		Delete("outbox_events").
		Where(squirrel.And{
			squirrel.Eq{"processed": true},
			squirrel.LtOrEq{"processed_at": cutoffTime},
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

// scanOutboxEvents — вспомогательная функция для сканирования строк
func scanOutboxEvents(op string, rows pgx.Rows) ([]*entity.OutboxEvent, error) {
	var events []*entity.OutboxEvent

	for rows.Next() {
		var e entity.OutboxEvent
		err := rows.Scan(
			&e.ID,
			&e.EventType,
			&e.AggregateID,
			&e.Payload,
			&e.OccurredAt,
			&e.CreatedAt,
			&e.Processed,
			&e.ProcessedAt,
			&e.Error,
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

// executor — возвращает исполнителя запросов (поддержка транзакций)
func (r *OutboxRepository) executor(ctx context.Context) postgres.QueryExecuter {
	if qe, ok := transaction.TxFromCtx(ctx); ok {
		return qe
	}
	return r.storage
}
