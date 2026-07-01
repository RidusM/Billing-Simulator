package repository

import (
	"context"
	"fmt"
	"time"

	"bill-stripe-sim/pkg/storage/redis"
)

type CacheRepository struct {
	storage *redis.Redis
}

func NewCacheRepository(storage *redis.Redis) *CacheRepository {
	return &CacheRepository{
		storage: storage,
	}
}

func (r *CacheRepository) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	err := r.storage.Client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		return fmt.Errorf("cache.Set: %w", err)
	}
	return nil
}

func (r *CacheRepository) Get(ctx context.Context, key string, dest any) error {
	err := r.storage.Client.Get(ctx, key).Scan(dest)
	if err != nil {
		if redis.IsNil(err) {
			return nil
		}
		return fmt.Errorf("cache.Get: %w", err)
	}
	return nil
}

func (r *CacheRepository) Delete(ctx context.Context, key string) error {
	err := r.storage.Client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("cache.Delete: %w", err)
	}
	return nil
}

func (r *CacheRepository) Lock(ctx context.Context, key string, ttl time.Duration) (func(), error) {
	const op = "repository.cache.Lock"

	lockKey := fmt.Sprintf("lock:%s", key)

	ok, err := r.storage.Client.SetNX(ctx, lockKey, "locked", ttl).Result()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if !ok {
		return nil, fmt.Errorf("%s: lock already held for key %s", op, key)
	}

	release := func() {
		_ = r.storage.Client.Del(context.Background(), lockKey).Err()
	}

	return release, nil
}
