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

type SubscriptionRepository struct {
	storage *postgres.Postgres
}

func NewSubscriptionRepository(storage *postgres.Postgres) *SubscriptionRepository {
	return &SubscriptionRepository{
		storage: storage,
	}
}

func (r *SubscriptionRepository) Create(ctx context.Context, s *entity.Subscription) error {
	const op = "repository.subscription.Create"

	sql, args, err := r.storage.Builder.
		Insert("subscriptions").
		Columns("id", "public_id", "customer_id", "status", "price_id", "current_period_start", "current_period_end", "next_billing_at", "canceled_at", "created_at").
		Values(s.ID, s.PublicID, s.CustomerID, s.Status, s.PriceID, s.CurrentPeriodStart, s.CurrentPeriodEnd, s.NextBillingAt, s.CanceledAt, s.CreatedAt).
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

func (r *SubscriptionRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Subscription, error) {
	return r.findFirst(ctx, "repository.subscription.GetByID", squirrel.Eq{"id": id})
}

func (r *SubscriptionRepository) GetByPublicID(ctx context.Context, publicID string) (*entity.Subscription, error) {
	return r.findFirst(ctx, "repository.subscription.GetByPublicID", squirrel.Eq{"public_id": publicID})
}

func (r *SubscriptionRepository) findFirst(ctx context.Context, op string, filter any) (*entity.Subscription, error) {
	sql, args, err := r.storage.
		Select(
			"id",
			"public_id",
			"customer_id",
			"status",
			"price_id",
			"current_period_start",
			"current_period_end",
			"next_billing_at",
			"canceled_at",
			"created_at",
		).
		From("subscriptions").
		Where(filter).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var s entity.Subscription
	err = r.executor(ctx).QueryRow(ctx, sql, args...).Scan(
		&s.ID, &s.PublicID, &s.CustomerID, &s.Status, &s.PriceID,
		&s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.NextBillingAt, &s.CanceledAt, &s.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, entity.ErrSubscriptionNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &s, nil
}

func (r *SubscriptionRepository) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status entity.SubscriptionStatus,
) error {
	const op = "repository.subscription.UpdateStatus"

	sql, args, err := r.storage.
		Update("subscriptions").
		Set("status", status).
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
		return fmt.Errorf("%s: %w", op, entity.ErrSubscriptionNotFound)
	}

	return nil
}

func (r *SubscriptionRepository) UpdateNextBilling(
	ctx context.Context,
	id uuid.UUID,
	nextBilling time.Time,
	periodStart time.Time,
	periodEnd time.Time,
) error {
	const op = "repository.subscription.UpdateNextBilling"

	sql, args, err := r.storage.
		Update("subscriptions").
		Set("next_billing_at", nextBilling).
		Set("current_period_start", periodStart).
		Set("current_period_end", periodEnd).
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
		return fmt.Errorf("%s: %w", op, entity.ErrSubscriptionNotFound)
	}

	return nil
}

func (r *SubscriptionRepository) GetActiveForRenewal(
	ctx context.Context,
	currentTime time.Time,
) ([]*entity.Subscription, error) {
	const op = "repository.subscription.GetActiveForRenewal"

	sql, args, err := r.storage.
		Select(
			"id",
			"public_id",
			"customer_id",
			"status",
			"price_id",
			"current_period_start",
			"current_period_end",
			"next_billing_at",
			"canceled_at",
			"created_at",
		).
		From("subscriptions").
		Where(squirrel.And{
			squirrel.Eq{"status": entity.SubscriptionStatusActive},
			squirrel.LtOrEq{"next_billing_at": currentTime},
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := r.executor(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var subs []*entity.Subscription
	for rows.Next() {
		var s entity.Subscription
		err = rows.Scan(
			&s.ID,
			&s.PublicID,
			&s.CustomerID,
			&s.Status,
			&s.PriceID,
			&s.CurrentPeriodStart,
			&s.CurrentPeriodEnd,
			&s.NextBillingAt,
			&s.CanceledAt,
			&s.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		subs = append(subs, &s)
	}

	return subs, nil
}

func (r *SubscriptionRepository) executor(ctx context.Context) postgres.QueryExecuter {
	if qe, ok := transaction.TxFromCtx(ctx); ok {
		return qe
	}
	return r.storage
}
