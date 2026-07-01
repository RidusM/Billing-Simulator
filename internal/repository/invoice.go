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

type InvoiceRepository struct {
	storage *postgres.Postgres
}

func NewInvoiceRepository(storage *postgres.Postgres) *InvoiceRepository {
	return &InvoiceRepository{
		storage: storage,
	}
}

func (r *InvoiceRepository) Create(ctx context.Context, i *entity.Invoice) error {
	const op = "repository.invoice.Create"

	sql, args, err := r.storage.Builder.
		Insert("invoices").
		Columns("id", "public_id", "subscription_id", "customer_id", "amount", "currency", "status", "attempt_count", "created_at").
		Values(i.ID, i.PublicID, i.SubscriptionID, i.CustomerID, i.Amount, i.Currency, i.Status, i.AttemptCount, i.CreatedAt).
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

func (r *InvoiceRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Invoice, error) {
	return r.findFirst(ctx, "repository.invoice.GetByID", squirrel.Eq{"id": id})
}

func (r *InvoiceRepository) GetByPublicID(ctx context.Context, publicID string) (*entity.Invoice, error) {
	return r.findFirst(ctx, "repository.invoice.GetByPublicID", squirrel.Eq{"public_id": publicID})
}

func (r *InvoiceRepository) findFirst(ctx context.Context, op string, filter any) (*entity.Invoice, error) {
	sql, args, err := r.storage.
		Select(
			"id",
			"public_id",
			"subscription_id",
			"customer_id",
			"amount",
			"currency",
			"status",
			"attempt_count",
			"created_at",
		).
		From("invoices").
		Where(filter).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var i entity.Invoice
	err = r.executor(ctx).QueryRow(ctx, sql, args...).Scan(
		&i.ID,
		&i.PublicID,
		&i.SubscriptionID,
		&i.CustomerID,
		&i.Amount,
		&i.Currency,
		&i.Status,
		&i.AttemptCount,
		&i.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, entity.ErrInvoiceNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &i, nil
}

func (r *InvoiceRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entity.InvoiceStatus) error {
	const op = "repository.invoice.UpdateStatus"

	sql, args, err := r.storage.
		Update("invoices").
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
		return fmt.Errorf("%s: %w", op, entity.ErrInvoiceNotFound)
	}

	return nil
}

func (r *InvoiceRepository) IncrementAttempt(ctx context.Context, id uuid.UUID) error {
	const op = "repository.invoice.IncrementAttempt"

	sql, args, err := r.storage.
		Update("invoices").
		Set("attempt_count", squirrel.Expr("attempt_count + 1")).
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
		return fmt.Errorf("%s: %w", op, entity.ErrInvoiceNotFound)
	}

	return nil
}

func (r *InvoiceRepository) GetByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*entity.Invoice, error) {
	const op = "repository.invoice.GetByCustomerID"

	sql, args, err := r.storage.
		Select(
			"id",
			"public_id",
			"subscription_id",
			"customer_id",
			"amount",
			"currency",
			"status",
			"attempt_count",
			"created_at",
		).
		From("invoices").
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

	var invoices []*entity.Invoice
	for rows.Next() {
		var i entity.Invoice
		err = rows.Scan(
			&i.ID,
			&i.PublicID,
			&i.SubscriptionID,
			&i.CustomerID,
			&i.Amount,
			&i.Currency,
			&i.Status,
			&i.AttemptCount,
			&i.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		invoices = append(invoices, &i)
	}

	return invoices, nil
}

func (r *InvoiceRepository) executor(ctx context.Context) postgres.QueryExecuter {
	if qe, ok := transaction.TxFromCtx(ctx); ok {
		return qe
	}
	return r.storage
}
