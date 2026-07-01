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

type CustomerRepository struct {
	storage *postgres.Postgres
}

func NewCustomerRepository(storage *postgres.Postgres) *CustomerRepository {
	return &CustomerRepository{
		storage: storage,
	}
}

func (r *CustomerRepository) Create(ctx context.Context, c *entity.Customer) error {
	const op = "repository.customer.Create"

	sql, args, err := r.storage.Builder.
		Insert("customers").
		Columns("id", "public_id", "email", "created_at").
		Values(c.ID, c.PublicID, c.Email, c.CreatedAt).
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

func (r *CustomerRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Customer, error) {
	const op = "repository.customer.GetByID"

	sql, args, err := r.storage.
		Select("id", "public_id", "email", "created_at").
		From("customers").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var c entity.Customer
	err = r.executor(ctx).QueryRow(ctx, sql, args...).Scan(
		&c.ID,
		&c.PublicID,
		&c.Email,
		&c.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, entity.ErrCustomerNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &c, nil
}

func (r *CustomerRepository) GetByPublicID(ctx context.Context, publicID string) (*entity.Customer, error) {
	const op = "repository.customer.GetByPublicID"

	sql, args, err := r.storage.
		Select("id", "public_id", "email", "created_at").
		From("customers").
		Where(squirrel.Eq{"public_id": publicID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var c entity.Customer
	err = r.executor(ctx).QueryRow(ctx, sql, args...).Scan(
		&c.ID,
		&c.PublicID,
		&c.Email,
		&c.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, entity.ErrCustomerNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &c, nil
}

func (r *CustomerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const op = "repository.customer.Delete"

	sql, args, err := r.storage.
		Delete("customers").
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
		return fmt.Errorf("%s: %w", op, entity.ErrCustomerNotFound)
	}

	return nil
}

func (r *CustomerRepository) executor(ctx context.Context) postgres.QueryExecuter {
	if qe, ok := transaction.TxFromCtx(ctx); ok {
		return qe
	}
	return r.storage
}
