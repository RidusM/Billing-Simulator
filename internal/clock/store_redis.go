package clock

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"bill-stripe-sim/pkg/storage/redis"
)

const clockOffsetKey = "virtual_clock_offset"

type redisStore struct {
	storage *redis.Redis
}

func NewRedisStore(storage *redis.Redis) Store {
	return &redisStore{
		storage: storage,
	}
}

func (r *redisStore) SaveOffset(ctx context.Context, offset time.Duration) error {
	err := r.storage.Client.Set(ctx, clockOffsetKey, int64(offset), 0).Err()
	if err != nil {
		return fmt.Errorf("clock.redisStore.SaveOffset: %w", err)
	}
	return nil
}

func (r *redisStore) LoadOffset(ctx context.Context) (time.Duration, error) {
	const op = "clock.redisStore.LoadOffset"

	val, err := r.storage.Client.Get(ctx, clockOffsetKey).Result()
	if err != nil {
		if redis.IsNil(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	offsetNano, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return time.Duration(offsetNano), nil
}
