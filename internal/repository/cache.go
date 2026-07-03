package repository

import (
	"context"
	"encoding/json"
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
	const op = "repository.cache.Set"

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if err := r.storage.Client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *CacheRepository) SetBatch(ctx context.Context, items map[string]any, ttl time.Duration) error {
	pipe := r.storage.Client.Pipeline()
	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("cache.SetBatch marshal key=%s: %w", key, err)
		}
		pipe.Set(ctx, key, data, ttl)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("cache.SetBatch: %w", err)
	}
	return nil
}

func (r *CacheRepository) Get(ctx context.Context, key string, dest any) error {
	data, err := r.storage.Client.Get(ctx, key).Bytes()
	if err != nil {
		if redis.IsNil(err) {
			return nil
		}
		return fmt.Errorf("cache.Get: %w", err)
	}
	return json.Unmarshal(data, dest)
}

func (r *CacheRepository) Delete(ctx context.Context, key string) error {
	if err := r.storage.Client.Del(ctx, key).Err(); err != nil {
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
	return func() {
		_ = r.storage.Client.Del(context.Background(), lockKey).Err()
	}, nil
}
