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
		Columns("id", "public_id", "subscription_id", "customer_id", "amount", "amount_paid", "amount_remaining", "currency", "status", "period_start", "period_end", "due_date", "attempt_count", "attempted_at", "hosted_invoice_url", "invoice_pdf_url", "metadata", "created_at", "updated_at").
		Values(i.ID, i.PublicID, i.SubscriptionID, i.CustomerID, i.Amount, i.AmountPaid, i.AmountRemaining, i.Currency, i.Status, i.PeriodStart, i.PeriodEnd, i.DueDate, i.AttemptCount, i.AttemptedAt, i.HostedInvoiceURL, i.InvoicePDFURL, i.Metadata, i.CreatedAt, i.UpdatedAt).
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

func (r *InvoiceRepository) GetByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*entity.Invoice, error) {
	const op = "repository.invoice.GetByCustomerID"

	sql, args, err := r.storage.
		Select(
			"id", "public_id", "subscription_id", "customer_id", "amount", "amount_paid", "amount_remaining",
			"currency", "status", "period_start", "period_end", "due_date", "attempt_count", "attempted_at",
			"hosted_invoice_url", "invoice_pdf_url", "metadata", "created_at", "updated_at",
		).
		From("invoices").
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

	return scanInvoices(op, rows)
}

func (r *InvoiceRepository) GetBySubscriptionID(ctx context.Context, subscriptionID uuid.UUID) ([]*entity.Invoice, error) {
	const op = "repository.invoice.GetBySubscriptionID"

	sql, args, err := r.storage.
		Select(
			"id", "public_id", "subscription_id", "customer_id", "amount", "amount_paid", "amount_remaining",
			"currency", "status", "period_start", "period_end", "due_date", "attempt_count", "attempted_at",
			"hosted_invoice_url", "invoice_pdf_url", "metadata", "created_at", "updated_at",
		).
		From("invoices").
		Where(squirrel.And{
			squirrel.Eq{"subscription_id": subscriptionID},
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

	return scanInvoices(op, rows)
}

func (r *InvoiceRepository) GetByStatus(ctx context.Context, status entity.InvoiceStatus, limit uint64) ([]*entity.Invoice, error) {
	const op = "repository.invoice.GetByStatus"
	sql, args, err := r.storage.
		Select(
			"id", "public_id", "subscription_id", "customer_id", "amount", "amount_paid", "amount_remaining",
			"currency", "status", "period_start", "period_end", "due_date", "attempt_count", "attempted_at",
			"hosted_invoice_url", "invoice_pdf_url", "metadata", "created_at", "updated_at",
		).
		From("invoices").
		Where(squirrel.And{
			squirrel.Eq{"status": status},
			squirrel.Expr("deleted_at IS NULL"),
		}).
		OrderBy("created_at ASC").
		Limit(limit).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := r.executor(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	return scanInvoices(op, rows)
}

func (r *InvoiceRepository) GetOverdue(ctx context.Context, before time.Time, limit uint64) ([]*entity.Invoice, error) {
	const op = "repository.invoice.GetOverdue"
	sql, args, err := r.storage.
		Select(
			"id", "public_id", "subscription_id", "customer_id", "amount", "amount_paid", "amount_remaining",
			"currency", "status", "period_start", "period_end", "due_date", "attempt_count", "attempted_at",
			"hosted_invoice_url", "invoice_pdf_url", "metadata", "created_at", "updated_at",
		).
		From("invoices").
		Where(squirrel.And{
			squirrel.Eq{"status": entity.InvoiceStatusOpen},
			squirrel.Lt{"due_date": before},
			squirrel.Expr("deleted_at IS NULL"),
		}).
		OrderBy("due_date ASC").
		Limit(limit).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := r.executor(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	return scanInvoices(op, rows)
}

func (r *InvoiceRepository) Update(ctx context.Context, i *entity.Invoice) error {
	const op = "repository.invoice.Update"

	sql, args, err := r.storage.Builder.
		Update("invoices").
		Set("status", i.Status).
		Set("amount_paid", i.AmountPaid).
		Set("amount_remaining", i.AmountRemaining).
		Set("attempt_count", i.AttemptCount).
		Set("attempted_at", i.AttemptedAt).
		Set("hosted_invoice_url", i.HostedInvoiceURL).
		Set("invoice_pdf_url", i.InvoicePDFURL).
		Set("metadata", i.Metadata).
		Set("updated_at", i.UpdatedAt).
		Where(squirrel.Eq{"id": i.ID}).
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

func (r *InvoiceRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entity.InvoiceStatus) error {
	const op = "repository.invoice.UpdateStatus"

	sql, args, err := r.storage.Builder.
		Update("invoices").
		Set("status", status).
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
		return fmt.Errorf("%s: %w", op, entity.ErrInvoiceNotFound)
	}

	return nil
}

func (r *InvoiceRepository) IncrementAttempt(ctx context.Context, id uuid.UUID) error {
	const op = "repository.invoice.IncrementAttempt"

	now := time.Now().UTC()
	sql, args, err := r.storage.Builder.
		Update("invoices").
		Set("attempt_count", squirrel.Expr("attempt_count + 1")).
		Set("attempted_at", now).
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
		return fmt.Errorf("%s: %w", op, entity.ErrInvoiceNotFound)
	}

	return nil
}

func (r *InvoiceRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	const op = "repository.invoice.SoftDelete"

	sql, args, err := r.storage.Builder.
		Update("invoices").
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
		return fmt.Errorf("%s: %w", op, entity.ErrInvoiceNotFound)
	}

	return nil
}

func (r *InvoiceRepository) findFirst(ctx context.Context, op string, filter squirrel.Sqlizer) (*entity.Invoice, error) {
	sql, args, err := r.storage.
		Select(
			"id", "public_id", "subscription_id", "customer_id", "amount", "amount_paid", "amount_remaining",
			"currency", "status", "period_start", "period_end", "due_date", "attempt_count", "attempted_at",
			"hosted_invoice_url", "invoice_pdf_url", "metadata", "created_at", "updated_at",
		).
		From("invoices").
		Where(squirrel.And{filter, squirrel.Expr("deleted_at IS NULL")}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var i entity.Invoice
	err = r.executor(ctx).QueryRow(ctx, sql, args...).Scan(
		&i.ID, &i.PublicID, &i.SubscriptionID, &i.CustomerID, &i.Amount, &i.AmountPaid, &i.AmountRemaining,
		&i.Currency, &i.Status, &i.PeriodStart, &i.PeriodEnd, &i.DueDate, &i.AttemptCount, &i.AttemptedAt,
		&i.HostedInvoiceURL, &i.InvoicePDFURL, &i.Metadata, &i.CreatedAt, &i.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, entity.ErrInvoiceNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &i, nil
}

func scanInvoices(op string, rows pgx.Rows) ([]*entity.Invoice, error) {
	var invoices []*entity.Invoice
	for rows.Next() {
		var i entity.Invoice
		err := rows.Scan(
			&i.ID, &i.PublicID, &i.SubscriptionID, &i.CustomerID, &i.Amount, &i.AmountPaid, &i.AmountRemaining,
			&i.Currency, &i.Status, &i.PeriodStart, &i.PeriodEnd, &i.DueDate, &i.AttemptCount, &i.AttemptedAt,
			&i.HostedInvoiceURL, &i.InvoicePDFURL, &i.Metadata, &i.CreatedAt, &i.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		invoices = append(invoices, &i)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return invoices, nil
}

func (r *InvoiceRepository) executor(ctx context.Context) postgres.QueryExecuter {
	if qe, ok := transaction.TxFromCtx(ctx); ok {
		return qe
	}
	return r.storage
}
