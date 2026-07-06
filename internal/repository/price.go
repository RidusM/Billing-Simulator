package repository

import (
	"context"
	"errors"
	"fmt"

	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/storage/postgres"
	"bill-stripe-sim/pkg/storage/postgres/transaction"

	"github.com/Masterminds/squirrel"
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
		Columns("id", "amount", "currency", "interval", "interval_count", "created_at").
		Values(p.ID, p.Amount, p.Currency, p.Interval, p.IntervalCount, p.CreatedAt).
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

func (r *PriceRepository) GetByID(ctx context.Context, id string) (*entity.Price, error) {
	const op = "repository.price.GetByID"
	sql, args, err := r.storage.
		Select("id", "amount", "currency", "interval", "interval_count", "created_at").
		From("prices").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var p entity.Price
	err = r.executor(ctx).QueryRow(ctx, sql, args...).Scan(
		&p.ID,
		&p.Amount,
		&p.Currency,
		&p.Interval,
		&p.IntervalCount,
		&p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, entity.ErrInvalidPrice)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &p, nil
}

func (r *PriceRepository) List(ctx context.Context) ([]*entity.Price, error) {
	const op = "repository.price.List"
	sql, args, err := r.storage.
		Select("id", "amount", "currency", "interval", "interval_count", "created_at").
		From("prices").
		OrderBy("created_at ASC").
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
			&p.ID,
			&p.Amount,
			&p.Currency,
			&p.Interval,
			&p.IntervalCount,
			&p.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		prices = append(prices, &p)
	}
	return prices, nil
}

func (r *PriceRepository) executor(ctx context.Context) postgres.QueryExecuter {
	if qe, ok := transaction.TxFromCtx(ctx); ok {
		return qe
	}
	return r.storage
}
