package clock

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"bill-stripe-sim/pkg/storage/redis"
)

const (
	clockOffsetKey     = "virtual_clock_offset"
	clockPubSubChannel = "clock:offset_changed"
)

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

func (r *redisStore) PublishOffsetChanged(ctx context.Context, offset time.Duration) error {
	const op = "clock.redisStore.PublishOffsetChanged"
	err := r.storage.Client.Publish(ctx, clockPubSubChannel, int64(offset)).Err()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *redisStore) SubscribeOffsetChanged(ctx context.Context) (<-chan time.Duration, error) {
	const op = "clock.redisStore.SubscribeOffsetChanged"
	pubsub := r.storage.Client.Subscribe(ctx, clockPubSubChannel)

	ch := make(chan time.Duration, 100)

	go func() {
		defer close(ch)
		defer pubsub.Close()

		redisCh := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-redisCh:
				if !ok {
					return
				}
				offset, err := strconv.ParseInt(msg.Payload, 10, 64)
				if err != nil {
					continue
				}
				select {
				case ch <- time.Duration(offset):
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch, nil
}
