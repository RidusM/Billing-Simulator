package postgres

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"bill-stripe-sim/pkg/logger"

	"github.com/Masterminds/squirrel"
	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	_backoffMultiplier  = 2
	_defaultPingTimeout = 5 * time.Second
)

type Postgres struct {
	Builder squirrel.StatementBuilderType
	Pool    *pgxpool.Pool
	logger  logger.Logger
}

func New(baseCfg Config, log logger.Logger, opts ...Option) (*Postgres, error) {
	const op = "storage.postgres.New"

	cfg := defaultConfigs(baseCfg)
	for _, opt := range opts {
		opt(cfg)
	}
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	pg := &Postgres{
		logger:  log,
		Builder: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	poolConfig.MaxConns = cfg.MaxPoolSize

	poolConfig.ConnConfig.Tracer = otelpgx.NewTracer()

	currentBackoff := cfg.BaseRetryDelay

	for attemptCount := 1; attemptCount <= cfg.ConnAttempts; attemptCount++ {
		pg.Pool, err = pgxpool.NewWithConfig(context.Background(), poolConfig)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), _defaultPingTimeout)
			err = pg.Pool.Ping(ctx)
			cancel()

			if err == nil {
				pg.logger.Info("postgresql connection successful")
				return pg, nil
			}

			pg.Pool.Close()
		}

		//nolint:gosec // weak random is completely fine for exponential backoff jitter
		jitter := min(time.Duration(rand.Int64N(int64(currentBackoff*_backoffMultiplier))), cfg.MaxRetryDelay)

		pg.logger.Warn("postgresql connection attempt failed, retrying...",
			"operation", op,
			"attempt", attemptCount,
			"retry_after", jitter.String(),
			"error", err.Error(),
		)

		time.Sleep(jitter)
		if currentBackoff < cfg.MaxRetryDelay/2 {
    currentBackoff *= _backoffMultiplier
} else {
    currentBackoff = cfg.MaxRetryDelay
}
	}

	return nil, fmt.Errorf("%s: failed to connect after %d attempts: %w", op, cfg.ConnAttempts, err)
}

func (p *Postgres) Ping(ctx context.Context) error {
	if err := p.Pool.Ping(ctx); err != nil {
		return fmt.Errorf("storage.postgres.Ping: %w", err)
	}
	return nil
}

func (p *Postgres) Close() {
	if p.Pool != nil {
		p.logger.Info("closing postgresql connection pool...")
		p.Pool.Close()
		p.logger.Info("postgresql connection pool closed")
	}
}

func (p *Postgres) Select(columns ...string) squirrel.SelectBuilder {
	return p.Builder.Select(columns...)
}
func (p *Postgres) Insert(into string) squirrel.InsertBuilder  { return p.Builder.Insert(into) }
func (p *Postgres) Update(table string) squirrel.UpdateBuilder { return p.Builder.Update(table) }
func (p *Postgres) Delete(from string) squirrel.DeleteBuilder  { return p.Builder.Delete(from) }
