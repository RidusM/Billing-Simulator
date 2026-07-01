package redis

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

const _defaultPingTimeout = 5 * time.Second

type Redis struct {
	Client   *redis.Client
	CacheTTL time.Duration
}

func New(baseCfg Config, opts ...Option) (*Redis, error) {
	const op = "storage.redis.New"

	cfg := defaultConfigs(baseCfg)

	for _, opt := range opts {
		opt(cfg)
	}

	clientOpts := &redis.Options{
		Addr:         net.JoinHostPort(cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		PoolTimeout:  cfg.PoolTimeout,
	}

	rdb := &Redis{
		CacheTTL: cfg.TTL,
	}
	rdb.Client = redis.NewClient(clientOpts)

	ctx, cancel := context.WithTimeout(context.Background(), _defaultPingTimeout)
	defer cancel()

	if err := rdb.Client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := redisotel.InstrumentTracing(rdb.Client); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if err := redisotel.InstrumentMetrics(rdb.Client); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return rdb, nil
}

func (r *Redis) Close() error {
	if err := r.Client.Close(); err != nil {
		return fmt.Errorf("storage.redis.Close: %w", err)
	}
	return nil
}

func IsNil(err error) bool {
	return errors.Is(err, redis.Nil)
}
