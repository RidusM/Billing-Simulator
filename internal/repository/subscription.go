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
		Columns("id", "public_id", "customer_id", "price_id", "status", "current_period_start", "current_period_end", "next_billing_at", "trial_start", "trial_end", "canceled_at", "cancel_at_period_end", "metadata", "created_at", "updated_at").
		Values(s.ID, s.PublicID, s.CustomerID, s.PriceID, s.Status, s.CurrentPeriodStart, s.CurrentPeriodEnd, s.NextBillingAt, s.TrialStart, s.TrialEnd, s.CanceledAt, s.CancelAtPeriodEnd, s.Metadata, s.CreatedAt, s.UpdatedAt).
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

func (r *SubscriptionRepository) GetByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*entity.Subscription, error) {
	const op = "repository.subscription.GetByCustomerID"
	sql, args, err := r.storage.
		Select("id", "public_id", "customer_id", "price_id", "status", "current_period_start", "current_period_end", "next_billing_at", "trial_start", "trial_end", "canceled_at", "cancel_at_period_end", "metadata", "created_at", "updated_at").
		From("subscriptions").
		Where(squirrel.And{
			squirrel.Eq{"customer_id": customerID},
			squirrel.Expr("deleted_at IS NULL"),
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

	return scanSubscriptions(op, rows)
}

func (r *SubscriptionRepository) GetActiveForRenewal(ctx context.Context, currentTime time.Time) ([]*entity.Subscription, error) {
	const op = "repository.subscription.GetActiveForRenewal"

	sql, args, err := r.storage.
		Select(
			"id", "public_id", "customer_id", "price_id", "status", "current_period_start", "current_period_end",
			"next_billing_at", "trial_start", "trial_end", "canceled_at", "cancel_at_period_end", "metadata", "created_at", "updated_at",
		).
		From("subscriptions").
		Where(squirrel.And{
			squirrel.Expr("status IN ('active', 'trialing')"),
			squirrel.LtOrEq{"next_billing_at": currentTime},
			squirrel.Expr("deleted_at IS NULL"),
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

	return scanSubscriptions(op, rows)
}

func (r *SubscriptionRepository) Update(ctx context.Context, s *entity.Subscription) error {
	const op = "repository.subscription.Update"

	sql, args, err := r.storage.Builder.
		Update("subscriptions").
		Set("status", s.Status).
		Set("current_period_start", s.CurrentPeriodStart).
		Set("current_period_end", s.CurrentPeriodEnd).
		Set("next_billing_at", s.NextBillingAt).
		Set("trial_start", s.TrialStart).
		Set("trial_end", s.TrialEnd).
		Set("canceled_at", s.CanceledAt).
		Set("cancel_at_period_end", s.CancelAtPeriodEnd).
		Set("metadata", s.Metadata).
		Set("updated_at", s.UpdatedAt).
		Where(squirrel.Eq{"id": s.ID}).
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

func (r *SubscriptionRepository) UpdateNextBilling(ctx context.Context, id uuid.UUID, nextBilling, periodStart, periodEnd time.Time) error {
	const op = "repository.subscription.UpdateNextBilling"

	sql, args, err := r.storage.Builder.
		Update("subscriptions").
		Set("next_billing_at", nextBilling).
		Set("current_period_start", periodStart).
		Set("current_period_end", periodEnd).
		Set("updated_at", "NOW()").
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

func (r *SubscriptionRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	const op = "repository.subscription.SoftDelete"

	sql, args, err := r.storage.Builder.
		Update("subscriptions").
		Set("deleted_at", "NOW()").
		Where(squirrel.And{
			squirrel.Eq{"id": id},
			squirrel.Expr("deleted_at IS NULL"),
		}).
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

func (r *SubscriptionRepository) findFirst(ctx context.Context, op string, filter squirrel.Sqlizer) (*entity.Subscription, error) {
	sql, args, err := r.storage.
		Select(
			"id", "public_id", "customer_id", "price_id", "status", "current_period_start", "current_period_end",
			"next_billing_at", "trial_start", "trial_end", "canceled_at", "cancel_at_period_end", "metadata", "created_at", "updated_at",
		).
		From("subscriptions").
		Where(squirrel.And{filter, squirrel.Expr("deleted_at IS NULL")}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var s entity.Subscription
	err = r.executor(ctx).QueryRow(ctx, sql, args...).Scan(
		&s.ID, &s.PublicID, &s.CustomerID, &s.PriceID, &s.Status,
		&s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.NextBillingAt,
		&s.TrialStart, &s.TrialEnd, &s.CanceledAt, &s.CancelAtPeriodEnd,
		&s.Metadata, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, entity.ErrSubscriptionNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &s, nil
}

func scanSubscriptions(op string, rows pgx.Rows) ([]*entity.Subscription, error) {
	var subscriptions []*entity.Subscription
	for rows.Next() {
		var s entity.Subscription
		err := rows.Scan(
			&s.ID, &s.PublicID, &s.CustomerID, &s.PriceID, &s.Status,
			&s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.NextBillingAt,
			&s.TrialStart, &s.TrialEnd, &s.CanceledAt, &s.CancelAtPeriodEnd,
			&s.Metadata, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		subscriptions = append(subscriptions, &s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return subscriptions, nil
}

func (r *SubscriptionRepository) executor(ctx context.Context) postgres.QueryExecuter {
	if qe, ok := transaction.TxFromCtx(ctx); ok {
		return qe
	}
	return r.storage
}
