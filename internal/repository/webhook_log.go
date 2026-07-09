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

type WebhookLogRepository struct {
	storage *postgres.Postgres
}

func NewWebhookLogRepository(storage *postgres.Postgres) *WebhookLogRepository {
	return &WebhookLogRepository{storage: storage}
}

func (r *WebhookLogRepository) Create(ctx context.Context, wl *entity.WebhookLog) error {
	const op = "repository.webhook_log.Create"
	sql, args, err := r.storage.Builder.
		Insert("webhook_logs").
		Columns("id", "public_id", "event_id", "endpoint_id", "trace_id", "event_type", "payload", "target_url", "status", "attempt", "max_attempts", "next_attempt_at", "created_at", "updated_at").
		Values(wl.ID, wl.PublicID, wl.EventID, wl.EndpointID, wl.TraceID, wl.EventType, wl.Payload, wl.TargetURL, wl.Status, wl.Attempt, wl.MaxAttempts, wl.NextAttemptAt, wl.CreatedAt, wl.UpdatedAt).
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

func (r *WebhookLogRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.WebhookLog, error) {
	const op = "repository.webhook_log.GetByID"
	sql, args, err := r.storage.
		Select("id", "public_id", "event_id", "endpoint_id", "trace_id", "event_type", "payload", "target_url", "status", "response_code", "response_body", "attempt", "max_attempts", "error_message", "next_attempt_at", "delivered_at", "created_at", "updated_at").
		From("webhook_logs").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var wl entity.WebhookLog
	err = r.executor(ctx).QueryRow(ctx, sql, args...).Scan(
		&wl.ID, &wl.PublicID, &wl.EventID, &wl.EndpointID, &wl.TraceID, &wl.EventType, &wl.Payload,
		&wl.TargetURL, &wl.Status, &wl.ResponseCode, &wl.ResponseBody, &wl.Attempt, &wl.MaxAttempts,
		&wl.ErrorMessage, &wl.NextAttemptAt, &wl.DeliveredAt, &wl.CreatedAt, &wl.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, entity.ErrDataNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &wl, nil
}

func (r *WebhookLogRepository) GetPendingForRetry(ctx context.Context, currentTime time.Time, limit int) ([]*entity.WebhookLog, error) {
	const op = "repository.webhook_log.GetPendingForRetry"
	sql, args, err := r.storage.
		Select("id", "public_id", "event_id", "endpoint_id", "trace_id", "event_type", "payload", "target_url", "status", "attempt", "max_attempts", "next_attempt_at", "created_at", "updated_at").
		From("webhook_logs").
		Where(squirrel.And{
			squirrel.Eq{"status": entity.WebhookStatusPending},
			squirrel.LtOrEq{"next_attempt_at": currentTime},
		}).
		OrderBy("next_attempt_at ASC").
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

	var logs []*entity.WebhookLog
	for rows.Next() {
		var wl entity.WebhookLog
		err = rows.Scan(
			&wl.ID, &wl.PublicID, &wl.EventID, &wl.EndpointID, &wl.TraceID, &wl.EventType, &wl.Payload,
			&wl.TargetURL, &wl.Status, &wl.Attempt, &wl.MaxAttempts, &wl.NextAttemptAt, &wl.CreatedAt, &wl.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		logs = append(logs, &wl)
	}
	return logs, nil
}

func (r *WebhookLogRepository) GetByTraceID(ctx context.Context, traceID uuid.UUID) ([]*entity.WebhookLog, error) {
	const op = "repository.webhook_log.GetByTraceID"
	sql, args, err := r.storage.
		Select("id", "public_id", "event_id", "endpoint_id", "trace_id", "event_type", "payload", "target_url", "status", "response_code", "attempt", "error_message", "next_attempt_at", "delivered_at", "created_at").
		From("webhook_logs").
		Where(squirrel.Eq{"trace_id": traceID}).
		OrderBy("created_at ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := r.executor(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var logs []*entity.WebhookLog
	for rows.Next() {
		var wl entity.WebhookLog
		err = rows.Scan(
			&wl.ID, &wl.PublicID, &wl.EventID, &wl.EndpointID, &wl.TraceID, &wl.EventType, &wl.Payload,
			&wl.TargetURL, &wl.Status, &wl.ResponseCode, &wl.Attempt, &wl.ErrorMessage, &wl.NextAttemptAt, &wl.DeliveredAt, &wl.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		logs = append(logs, &wl)
	}
	return logs, nil
}

func (r *WebhookLogRepository) Update(ctx context.Context, wl *entity.WebhookLog) error {
	const op = "repository.webhook_log.Update"
	sql, args, err := r.storage.Builder.
		Update("webhook_logs").
		Set("status", wl.Status).
		Set("response_code", wl.ResponseCode).
		Set("response_body", wl.ResponseBody).
		Set("attempt", wl.Attempt).
		Set("error_message", wl.ErrorMessage).
		Set("next_attempt_at", wl.NextAttemptAt).
		Set("delivered_at", wl.DeliveredAt).
		Set("updated_at", wl.UpdatedAt).
		Where(squirrel.Eq{"id": wl.ID}).
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

func (r *WebhookLogRepository) MarkDelivered(ctx context.Context, id uuid.UUID, responseCode int, deliveredAt time.Time) error {
	const op = "repository.webhook_log.MarkDelivered"
	sql, args, err := r.storage.Builder.
		Update("webhook_logs").
		Set("status", entity.WebhookStatusDelivered).
		Set("response_code", responseCode).
		Set("delivered_at", deliveredAt).
		Set("updated_at", deliveredAt).
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

func (r *WebhookLogRepository) MarkFailed(ctx context.Context, id uuid.UUID, errorMessage string, nextAttemptAt time.Time, now time.Time) error {
	const op = "repository.webhook_log.MarkFailed"
	sql, args, err := r.storage.Builder.
		Update("webhook_logs").
		Set("status", entity.WebhookStatusFailed).
		Set("error_message", errorMessage).
		Set("next_attempt_at", nextAttemptAt).
		Set("attempt", squirrel.Expr("attempt + 1")).
		Set("updated_at", now).
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

func (r *WebhookLogRepository) DeleteOldLogs(ctx context.Context, olderThan time.Duration) (int64, error) {
	const op = "repository.webhook_log.DeleteOldLogs"
	sql, args, err := r.storage.Builder.
		Delete("webhook_logs").
		Where(squirrel.Expr("created_at <= NOW() - ? * INTERVAL '1 SECOND'", olderThan.Seconds())).
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

func (r *WebhookLogRepository) executor(ctx context.Context) postgres.QueryExecuter {
	if qe, ok := transaction.TxFromCtx(ctx); ok {
		return qe
	}
	return r.storage
}
