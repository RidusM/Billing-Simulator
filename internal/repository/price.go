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

type PriceRepository struct {
	storage *postgres.Postgres
}

func NewPriceRepository(storage *postgres.Postgres) *PriceRepository {
	return &PriceRepository{
		storage: storage,
	}
}

func (r *PriceRepository) Create(ctx context.Context, p *entity.Price) error {
	const op = "repository.price.Create"
	sql, args, err := r.storage.Builder.
		Insert("prices").
		Columns("id", "public_id", "product_id", "amount", "currency", "interval", "interval_count", "active", "metadata", "created_at", "updated_at").
		Values(p.ID, p.PublicID, p.ProductID, p.Amount, p.Currency, p.Interval, p.IntervalCount, p.Active, p.Metadata, p.CreatedAt, p.UpdatedAt).
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

func (r *PriceRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Price, error) {
	return r.findFirst(ctx, "repository.price.GetByID", squirrel.Eq{"id": id})
}

func (r *PriceRepository) GetByPublicID(ctx context.Context, publicID string) (*entity.Price, error) {
	return r.findFirst(ctx, "repository.price.GetByPublicID", squirrel.Eq{"public_id": publicID})
}

func (r *PriceRepository) GetByProductID(ctx context.Context, productID uuid.UUID) ([]*entity.Price, error) {
	const op = "repository.price.GetByProductID"
	sql, args, err := r.storage.
		Select("id", "public_id", "product_id", "amount", "currency", "interval", "interval_count", "active", "metadata", "created_at", "updated_at").
		From("prices").
		Where(squirrel.And{
			squirrel.Eq{"product_id": productID},
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

	var prices []*entity.Price
	for rows.Next() {
		var p entity.Price
		err = rows.Scan(
			&p.ID, &p.PublicID, &p.ProductID, &p.Amount, &p.Currency, &p.Interval, &p.IntervalCount,
			&p.Active, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		prices = append(prices, &p)
	}
	return prices, nil
}

func (r *PriceRepository) List(ctx context.Context, limit, offset int) ([]*entity.Price, error) {
	const op = "repository.price.List"
	sql, args, err := r.storage.
		Select("id", "public_id", "product_id", "amount", "currency", "interval", "interval_count", "active", "metadata", "created_at", "updated_at").
		From("prices").
		Where(squirrel.Expr("deleted_at IS NULL")).
		OrderBy("created_at ASC").
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

	var prices []*entity.Price
	for rows.Next() {
		var p entity.Price
		err = rows.Scan(
			&p.ID, &p.PublicID, &p.ProductID, &p.Amount, &p.Currency, &p.Interval, &p.IntervalCount,
			&p.Active, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		prices = append(prices, &p)
	}
	return prices, nil
}

func (r *PriceRepository) Update(ctx context.Context, p *entity.Price) error {
	const op = "repository.price.Update"
	sql, args, err := r.storage.Builder.
		Update("prices").
		Set("amount", p.Amount).
		Set("currency", p.Currency).
		Set("interval", p.Interval).
		Set("interval_count", p.IntervalCount).
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
		return fmt.Errorf("%s: %w", op, entity.ErrPriceNotFound)
	}
	return nil
}

func (r *PriceRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	const op = "repository.price.SoftDelete"
	sql, args, err := r.storage.Builder.
		Update("prices").
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
		return fmt.Errorf("%s: %w", op, entity.ErrPriceNotFound)
	}
	return nil
}

func (r *PriceRepository) findFirst(ctx context.Context, op string, filter squirrel.Sqlizer) (*entity.Price, error) {
	sql, args, err := r.storage.
		Select("id", "public_id", "product_id", "amount", "currency", "interval", "interval_count", "active", "metadata", "created_at", "updated_at").
		From("prices").
		Where(squirrel.And{filter, squirrel.Expr("deleted_at IS NULL")}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var p entity.Price
	err = r.executor(ctx).QueryRow(ctx, sql, args...).Scan(
		&p.ID, &p.PublicID, &p.ProductID, &p.Amount, &p.Currency, &p.Interval, &p.IntervalCount,
		&p.Active, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, entity.ErrPriceNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &p, nil
}

func (r *PriceRepository) executor(ctx context.Context) postgres.QueryExecuter {
	if qe, ok := transaction.TxFromCtx(ctx); ok {
		return qe
	}
	return r.storage
}
