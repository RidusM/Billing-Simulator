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

type ProductRepository struct {
	storage *postgres.Postgres
}

func NewProductRepository(storage *postgres.Postgres) *ProductRepository {
	return &ProductRepository{storage: storage}
}

func (r *ProductRepository) Create(ctx context.Context, p *entity.Product) error {
	const op = "repository.product.Create"
	sql, args, err := r.storage.Builder.
		Insert("products").
		Columns("id", "public_id", "name", "description", "active", "metadata", "created_at", "updated_at").
		Values(p.ID, p.PublicID, p.Name, p.Description, p.Active, p.Metadata, p.CreatedAt, p.UpdatedAt).
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

func (r *ProductRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Product, error) {
	return r.findFirst(ctx, "repository.product.GetByID", squirrel.Eq{"id": id})
}

func (r *ProductRepository) GetByPublicID(ctx context.Context, publicID string) (*entity.Product, error) {
	return r.findFirst(ctx, "repository.product.GetByPublicID", squirrel.Eq{"public_id": publicID})
}

func (r *ProductRepository) List(ctx context.Context, limit, offset int) ([]*entity.Product, error) {
	const op = "repository.product.List"
	sql, args, err := r.storage.
		Select("id", "public_id", "name", "description", "active", "metadata", "created_at", "updated_at").
		From("products").
		Where(squirrel.Expr("deleted_at IS NULL")).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := r.executor(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var products []*entity.Product
	for rows.Next() {
		var p entity.Product
		err = rows.Scan(&p.ID, &p.PublicID, &p.Name, &p.Description, &p.Active, &p.Metadata, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		products = append(products, &p)
	}
	return products, nil
}

func (r *ProductRepository) Update(ctx context.Context, p *entity.Product) error {
	const op = "repository.product.Update"
	sql, args, err := r.storage.Builder.
		Update("products").
		Set("name", p.Name).
		Set("description", p.Description).
		Set("active", p.Active).
		Set("metadata", p.Metadata).
		Set("updated_at", p.UpdatedAt).
		Where(squirrel.Eq{"id": p.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	tag, err := r.executor(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, entity.ErrProductNotFound)
	}
	return nil
}

func (r *ProductRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	const op = "repository.product.SoftDelete"
	sql, args, err := r.storage.Builder.
		Update("products").
		Set("deleted_at", "NOW()").
		Set("active", false).
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
		return fmt.Errorf("%s: %w", op, entity.ErrProductNotFound)
	}
	return nil
}

func (r *ProductRepository) findFirst(ctx context.Context, op string, filter squirrel.Sqlizer) (*entity.Product, error) {
	sql, args, err := r.storage.
		Select("id", "public_id", "name", "description", "active", "metadata", "created_at", "updated_at").
		From("products").
		Where(squirrel.And{filter, squirrel.Expr("deleted_at IS NULL")}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var p entity.Product
	err = r.executor(ctx).QueryRow(ctx, sql, args...).Scan(
		&p.ID, &p.PublicID, &p.Name, &p.Description, &p.Active, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, entity.ErrProductNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &p, nil
}

func (r *ProductRepository) executor(ctx context.Context) postgres.QueryExecuter {
	if qe, ok := transaction.TxFromCtx(ctx); ok {
		return qe
	}
	return r.storage
}
