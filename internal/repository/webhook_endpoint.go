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
	"github.com/jackc/pgx/v5/pgconn"
)

type WebhookEndpointRepository struct {
	storage *postgres.Postgres
}

func NewWebhookEndpointRepository(storage *postgres.Postgres) *WebhookEndpointRepository {
	return &WebhookEndpointRepository{
		storage: storage,
	}
}

func (r *WebhookEndpointRepository) Create(ctx context.Context, e *entity.WebhookEndpoint) error {
	const op = "repository.webhook_endpoint.Create"
	sql, args, err := r.storage.Builder.
		Insert("webhook_endpoints").
		Columns("id", "customer_id", "url", "secret", "enabled", "created_at", "updated_at").
		Values(e.ID, e.CustomerID, e.URL, e.Secret, e.Enabled, e.CreatedAt, e.UpdatedAt).
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

func (r *WebhookEndpointRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.WebhookEndpoint, error) {
	const op = "repository.webhook_endpoint.GetByID"
	sql, args, err := r.storage.
		Select("id", "customer_id", "url", "secret", "enabled", "created_at", "updated_at").
		From("webhook_endpoints").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var e entity.WebhookEndpoint
	err = r.executor(ctx).QueryRow(ctx, sql, args...).Scan(
		&e.ID,
		&e.CustomerID,
		&e.URL,
		&e.Secret,
		&e.Enabled,
		&e.CreatedAt,
		&e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, entity.ErrDataNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &e, nil
}

func (r *WebhookEndpointRepository) GetByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*entity.WebhookEndpoint, error) {
	const op = "repository.webhook_endpoint.GetByCustomerID"
	sql, args, err := r.storage.
		Select("id", "customer_id", "url", "secret", "enabled", "created_at", "updated_at").
		From("webhook_endpoints").
		Where(squirrel.Eq{"customer_id": customerID}).
		OrderBy("created_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := r.executor(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var endpoints []*entity.WebhookEndpoint
	for rows.Next() {
		var e entity.WebhookEndpoint
		err = rows.Scan(
			&e.ID,
			&e.CustomerID,
			&e.URL,
			&e.Secret,
			&e.Enabled,
			&e.CreatedAt,
			&e.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		endpoints = append(endpoints, &e)
	}
	return endpoints, nil
}

func (r *WebhookEndpointRepository) GetActiveByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*entity.WebhookEndpoint, error) {
	const op = "repository.webhook_endpoint.GetActiveByCustomerID"
	sql, args, err := r.storage.
		Select("id", "customer_id", "url", "secret", "enabled", "created_at", "updated_at").
		From("webhook_endpoints").
		Where(squirrel.And{
			squirrel.Eq{"customer_id": customerID},
			squirrel.Eq{"enabled": true},
		}).
		OrderBy("created_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := r.executor(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var endpoints []*entity.WebhookEndpoint
	for rows.Next() {
		var e entity.WebhookEndpoint
		err = rows.Scan(
			&e.ID,
			&e.CustomerID,
			&e.URL,
			&e.Secret,
			&e.Enabled,
			&e.CreatedAt,
			&e.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		endpoints = append(endpoints, &e)
	}
	return endpoints, nil
}

func (r *WebhookEndpointRepository) Update(ctx context.Context, e *entity.WebhookEndpoint) error {
	const op = "repository.webhook_endpoint.Update"
	sql, args, err := r.storage.Builder.
		Update("webhook_endpoints").
		Set("url", e.URL).
		Set("secret", e.Secret).
		Set("enabled", e.Enabled).
		Set("updated_at", e.UpdatedAt).
		Where(squirrel.Eq{"id": e.ID}).
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

func (r *WebhookEndpointRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const op = "repository.webhook_endpoint.Delete"
	sql, args, err := r.storage.Builder.
		Delete("webhook_endpoints").
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

func (r *WebhookEndpointRepository) executor(ctx context.Context) postgres.QueryExecuter {
	if qe, ok := transaction.TxFromCtx(ctx); ok {
		return qe
	}
	return r.storage
}
