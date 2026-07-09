package redis

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"bill-stripe-sim/pkg/logger"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

const (
	_defaultPingTimeout  = 5 * time.Second
	_defaultConnAttempts = 5
	_defaultRetryDelay   = 1 * time.Second
)

var (
	ErrRedisNotConnected = errors.New("redis not connected")
)

type Redis struct {
	Client   *redis.Client
	CacheTTL time.Duration
	log      logger.Logger
}

type StringCmd = redis.StringCmd

func New(baseCfg Config, log logger.Logger, opts ...Option) (*Redis, error) {
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
		log:      log,
	}

	rdb.Client = redis.NewClient(clientOpts)

	var lastErr error
	for attempt := 1; attempt <= _defaultConnAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), _defaultPingTimeout)
		err := rdb.Client.Ping(ctx).Err()
		cancel()

		if err == nil {
			log.Info("redis connection successful",
				"operation", op,
				"host", cfg.Host,
				"port", cfg.Port,
				"db", cfg.DB,
			)
			break
		}

		lastErr = err
		log.Warn("redis connection attempt failed, retrying...",
			"operation", op,
			"attempt", attempt,
			"max_attempts", _defaultConnAttempts,
			"error", err.Error(),
		)

		if attempt < _defaultConnAttempts {
			time.Sleep(_defaultRetryDelay)
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("%s: failed to connect after %d attempts: %w", op, _defaultConnAttempts, lastErr)
	}

	if err := redisotel.InstrumentTracing(rdb.Client); err != nil {
		rdb.log.LogAttrs(context.Background(), logger.WarnLevel, "failed to instrument redis tracing, continuing without it",
			logger.String("operation", op),
			logger.Error(err),
		)
	}
	if err := redisotel.InstrumentMetrics(rdb.Client); err != nil {
		rdb.log.LogAttrs(context.Background(), logger.WarnLevel, "failed to instrument redis metrics, continuing without it",
			logger.String("operation", op),
			logger.Error(err),
		)
	}

	return rdb, nil
}

func (r *Redis) Ping(ctx context.Context) error {
	if r.Client == nil {
		return ErrRedisNotConnected
	}
	if err := r.Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("storage.redis.Ping: %w", err)
	}
	return nil
}

func (r *Redis) Close() error {
	if r.Client == nil {
		return nil
	}
	r.log.Info("closing redis connection...")
	if err := r.Client.Close(); err != nil {
		return fmt.Errorf("storage.redis.Close: %w", err)
	}
	r.log.Info("redis connection closed")
	return nil
}

func IsNil(err error) bool {
	return errors.Is(err, redis.Nil)
}
