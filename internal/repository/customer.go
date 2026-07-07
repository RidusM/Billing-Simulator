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
		Columns("id", "public_id", "email", "name", "phone", "metadata", "created_at", "updated_at").
		Values(c.ID, c.PublicID, c.Email, c.Name, c.Phone, c.Metadata, c.CreatedAt, c.UpdatedAt).
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
	return r.findFirst(ctx, "repository.customer.GetByID", squirrel.Eq{"id": id})
}

func (r *CustomerRepository) GetByPublicID(ctx context.Context, publicID string) (*entity.Customer, error) {
	return r.findFirst(ctx, "repository.customer.GetByPublicID", squirrel.Eq{"public_id": publicID})
}

func (r *CustomerRepository) GetByEmail(ctx context.Context, email string) (*entity.Customer, error) {
	return r.findFirst(ctx, "repository.customer.GetByEmail", squirrel.Eq{"email": email})
}

func (r *CustomerRepository) Update(ctx context.Context, c *entity.Customer) error {
	const op = "repository.customer.Update"

	sql, args, err := r.storage.Builder.
		Update("customers").
		Set("email", c.Email).
		Set("name", c.Name).
		Set("phone", c.Phone).
		Set("metadata", c.Metadata).
		Set("updated_at", c.UpdatedAt).
		Where(squirrel.Eq{"id": c.ID}).
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

func (r *CustomerRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	const op = "repository.customer.SoftDelete"

	sql, args, err := r.storage.Builder.
		Update("customers").
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
		return fmt.Errorf("%s: %w", op, entity.ErrCustomerNotFound)
	}

	return nil
}

func (r *CustomerRepository) findFirst(ctx context.Context, op string, filter squirrel.Sqlizer) (*entity.Customer, error) {
	if eq, ok := filter.(squirrel.Eq); ok && len(eq) == 0 {
		return nil, fmt.Errorf("%s: empty filter", op)
	}

	sql, args, err := r.storage.
		Select("id", "public_id", "email", "name", "phone", "metadata", "created_at", "updated_at").
		From("customers").
		Where(squirrel.And{filter, squirrel.Expr("deleted_at IS NULL")}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var c entity.Customer
	err = r.executor(ctx).QueryRow(ctx, sql, args...).Scan(
		&c.ID,
		&c.PublicID,
		&c.Email,
		&c.Name,
		&c.Phone,
		&c.Metadata,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, entity.ErrCustomerNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &c, nil
}

func (r *CustomerRepository) executor(ctx context.Context) postgres.QueryExecuter {
	if qe, ok := transaction.TxFromCtx(ctx); ok {
		return qe
	}
	return r.storage
}
