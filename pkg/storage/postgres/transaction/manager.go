package transaction

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"bill-stripe-sim/pkg/logger"
	"bill-stripe-sim/pkg/storage/postgres"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const _backoffMultiplier = 2

type txKey struct{}

func TxFromCtx(ctx context.Context) (postgres.QueryExecuter, bool) {
	qe, ok := ctx.Value(txKey{}).(postgres.QueryExecuter)
	return qe, ok
}

type Manager struct {
	pool   *postgres.Postgres
	logger logger.Logger
	cfg    *Config
}

func NewManager(
	pool *postgres.Postgres,
	logger logger.Logger,
	opts ...Option,
) (*Manager, error) {
	cfg := defaultConfigs()
	for _, opt := range opts {
		opt(cfg)
	}

	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("storage.postgres.transaction.NewManager: %w", err)
	}

	return &Manager{
		pool:   pool,
		logger: logger,
		cfg:    cfg,
	}, nil
}

func (tm *Manager) ExecuteInTransaction(
	ctx context.Context,
	txName string,
	fn func(ctx context.Context) error,
) error {
	const op = "storage.postgres.transaction.ExecuteInTransaction"

	var lastErr error
	currentBackoff := tm.cfg.BaseRetryDelay

	for attempt := 1; attempt <= tm.cfg.MaxAttempts; attempt++ {
		err := tm.doTransaction(ctx, fn)
		if err == nil {
			return nil
		}

		lastErr = err

		if !isRetryableError(lastErr) || attempt == tm.cfg.MaxAttempts {
			return HandleError(lastErr)
		}

		//nolint:gosec // weak random is completely fine for exponential backoff jitter
		jitter := min(
			time.Duration(rand.Int64N(int64(currentBackoff*_backoffMultiplier))),
			tm.cfg.MaxRetryDelay,
		)

		tm.logger.LogAttrs(ctx, logger.WarnLevel, "retrying transaction",
			logger.String("op", op),
			logger.String("transaction", txName),
			logger.Int("attempt", attempt),
			logger.Int("max_attempts", tm.cfg.MaxAttempts),
			logger.String("retry_after", jitter.String()),
			logger.Any("error", lastErr.Error()),
		)

		timer := time.NewTimer(jitter)
		select {
		case <-timer.C:
			currentBackoff = min(currentBackoff*_backoffMultiplier, tm.cfg.MaxRetryDelay)
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("%s: %s: %w", op, txName, ctx.Err())
		}
		timer.Stop()
	}

	return fmt.Errorf("%s: %s: %w", op, txName, HandleError(lastErr))
}

func (tm *Manager) doTransaction(
	ctx context.Context,
	fn func(ctx context.Context) error,
) error {
	tx, err := tm.pool.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tm.safelyRollback(ctx, tx)

	txCtx := context.WithValue(ctx, txKey{}, &postgres.TxQueryExecuter{Tx: tx})

	if err = fn(txCtx); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}
	return nil
}

func (tm *Manager) safelyRollback(ctx context.Context, tx pgx.Tx) {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		tm.logger.LogAttrs(ctx, logger.ErrorLevel, "rollback failed",
			logger.Any("error", err.Error()),
		)
	}
}

func isRetryableError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "40P01", "40001", "08000", "08003", "08006", "08001", "08004", "08007", "08P01":
			return true
		}
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, pgx.ErrTxClosed) {
		return true
	}
	return false
}
