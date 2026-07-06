package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bill-stripe-sim/pkg/storage/redis"

	"github.com/google/uuid"
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
	const op = "repository.cache.Get"
	data, err := r.storage.Client.Get(ctx, key).Bytes()
	if err != nil {
		if redis.IsNil(err) {
			return fmt.Errorf("%s: cache miss: %w", op)
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("%s: unmarshal: %w", op, err)
	}
	return nil
}

func (r *CacheRepository) GetBatch(ctx context.Context, keys []string) (map[string][]byte, error) {
	const op = "repository.cache.GetBatch"
	pipe := r.storage.Client.Pipeline()
	cmds := make(map[string]*redis.StringCmd, len(keys))

	for _, key := range keys {
		cmds[key] = pipe.Get(ctx, key)
	}

	if _, err := pipe.Exec(ctx); err != nil && !redis.IsNil(err) {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	result := make(map[string][]byte, len(keys))
	for key, cmd := range cmds {
		if val, err := cmd.Result(); err == nil {
			result[key] = []byte(val)
		}
	}
	return result, nil
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
	lockValue := uuid.New().String()

	ok, err := r.storage.Client.SetNX(ctx, lockKey, lockValue, ttl).Result()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if !ok {
		return nil, fmt.Errorf("%s: lock already held for key %s", op, key)
	}

	return func() {
		var luaRelease = `
			if redis.call("get", KEYS[1]) == ARGV[1] then
				return redis.call("del", KEYS[1])
			else
				return 0
			end`
		_ = r.storage.Client.Eval(context.Background(), luaRelease, []string{lockKey}, lockValue).Err()
	}, nil
}

func (r *CacheRepository) Exists(ctx context.Context, key string) (bool, error) {
	const op = "repository.cache.Exists"
	exists, err := r.storage.Client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return exists > 0, nil
}

func (r *CacheRepository) Incr(ctx context.Context, key string) (int64, error) {
	const op = "repository.cache.Incr"
	val, err := r.storage.Client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return val, nil
}
