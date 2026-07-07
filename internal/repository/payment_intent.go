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

type PaymentIntentRepository struct {
	storage *postgres.Postgres
}

func NewPaymentIntentRepository(storage *postgres.Postgres) *PaymentIntentRepository {
	return &PaymentIntentRepository{storage: storage}
}

func (r *PaymentIntentRepository) Create(ctx context.Context, pi *entity.PaymentIntent) error {
	const op = "repository.payment_intent.Create"
	sql, args, err := r.storage.Builder.
		Insert("payment_intents").
		Columns("id", "public_id", "invoice_id", "customer_id", "amount", "amount_captured", "currency", "status", "last_payment_error", "payment_method_id", "payment_method_type", "metadata", "created_at", "updated_at").
		Values(pi.ID, pi.PublicID, pi.InvoiceID, pi.CustomerID, pi.Amount, pi.AmountCaptured, pi.Currency, pi.Status, pi.LastPaymentError, pi.PaymentMethodID, pi.PaymentMethodType, pi.Metadata, pi.CreatedAt, pi.UpdatedAt).
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

func (r *PaymentIntentRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.PaymentIntent, error) {
	return r.findFirst(ctx, "repository.payment_intent.GetByID", squirrel.Eq{"id": id})
}

func (r *PaymentIntentRepository) GetByPublicID(ctx context.Context, publicID string) (*entity.PaymentIntent, error) {
	return r.findFirst(ctx, "repository.payment_intent.GetByPublicID", squirrel.Eq{"public_id": publicID})
}

func (r *PaymentIntentRepository) GetByInvoiceID(ctx context.Context, invoiceID uuid.UUID) (*entity.PaymentIntent, error) {
	return r.findFirst(ctx, "repository.payment_intent.GetByInvoiceID", squirrel.Eq{"invoice_id": invoiceID})
}

func (r *PaymentIntentRepository) Update(ctx context.Context, pi *entity.PaymentIntent) error {
	const op = "repository.payment_intent.Update"
	sql, args, err := r.storage.Builder.
		Update("payment_intents").
		Set("status", pi.Status).
		Set("amount_captured", pi.AmountCaptured).
		Set("last_payment_error", pi.LastPaymentError).
		Set("payment_method_id", pi.PaymentMethodID).
		Set("metadata", pi.Metadata).
		Set("updated_at", pi.UpdatedAt).
		Where(squirrel.Eq{"id": pi.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	tag, err := r.executor(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, entity.ErrPaymentIntentNotFound)
	}
	return nil
}

func (r *PaymentIntentRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	const op = "repository.payment_intent.SoftDelete"
	sql, args, err := r.storage.Builder.
		Update("payment_intents").
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
		return fmt.Errorf("%s: %w", op, entity.ErrPaymentIntentNotFound)
	}
	return nil
}

func (r *PaymentIntentRepository) findFirst(ctx context.Context, op string, filter squirrel.Sqlizer) (*entity.PaymentIntent, error) {
	sql, args, err := r.storage.
		Select("id", "public_id", "invoice_id", "customer_id", "amount", "amount_captured", "currency", "status", "last_payment_error", "payment_method_id", "payment_method_type", "metadata", "created_at", "updated_at").
		From("payment_intents").
		Where(squirrel.And{filter, squirrel.Expr("deleted_at IS NULL")}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var pi entity.PaymentIntent
	err = r.executor(ctx).QueryRow(ctx, sql, args...).Scan(
		&pi.ID, &pi.PublicID, &pi.InvoiceID, &pi.CustomerID, &pi.Amount, &pi.AmountCaptured,
		&pi.Currency, &pi.Status, &pi.LastPaymentError, &pi.PaymentMethodID, &pi.PaymentMethodType,
		&pi.Metadata, &pi.CreatedAt, &pi.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, entity.ErrPaymentIntentNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &pi, nil
}

func (r *PaymentIntentRepository) executor(ctx context.Context) postgres.QueryExecuter {
	if qe, ok := transaction.TxFromCtx(ctx); ok {
		return qe
	}
	return r.storage
}
