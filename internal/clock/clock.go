package clock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"bill-stripe-sim/pkg/logger"
)

type Store interface {
	SaveOffset(ctx context.Context, offset time.Duration) error
	LoadOffset(ctx context.Context) (time.Duration, error)
	PublishOffsetChanged(ctx context.Context, offset time.Duration) error
	SubscribeOffsetChanged(ctx context.Context) (<-chan time.Duration, error)
}

type TimeJumpListener func(oldTime, newTime time.Time)

type VirtualClock struct {
	mu        sync.RWMutex
	offset    time.Duration
	store     Store
	logger    logger.Logger
	listeners []TimeJumpListener
}

func NewVirtualClock(s Store, logger logger.Logger) (*VirtualClock, error) {
	vc := &VirtualClock{
		store:  s,
		logger: logger,
	}
	if s != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		offset, err := s.LoadOffset(ctx)
		if err != nil {
			return nil, fmt.Errorf("clock.NewVirtualClock: load offset: %w", err)
		}
		vc.offset = offset
		logger.Info("virtual clock initialized", "offset", offset.String())
	}
	return vc, nil
}

func (vc *VirtualClock) StartSync(ctx context.Context) error {
	if vc.store == nil {
		return nil
	}

	ch, err := vc.store.SubscribeOffsetChanged(ctx)
	if err != nil {
		return fmt.Errorf("clock.VirtualClock.StartSync: subscribe: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case newOffset, ok := <-ch:
				if !ok {
					return
				}
				vc.mu.Lock()
				oldOffset := vc.offset
				vc.offset = newOffset
				vc.mu.Unlock()

				if oldOffset != newOffset {
					vc.logger.Info("clock offset synced from another instance",
						"old_offset", oldOffset.String(),
						"new_offset", newOffset.String(),
					)
				}
			}
		}
	}()

	return nil
}

func (vc *VirtualClock) Now() time.Time {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return time.Now().Add(vc.offset)
}

func (vc *VirtualClock) OnTimeJump(listener TimeJumpListener) {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.listeners = append(vc.listeners, listener)
}

func (vc *VirtualClock) Advance(d time.Duration) error {
	vc.mu.Lock()
	realNow := time.Now()
	oldTime := realNow.Add(vc.offset)
	vc.offset += d
	newTime := realNow.Add(vc.offset)
	offset := vc.offset
	vc.mu.Unlock()

	if vc.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := vc.store.SaveOffset(ctx, offset); err != nil {
			vc.logger.Error("failed to save clock offset to Redis in Advance", "error", err)
			return fmt.Errorf("save offset: %w", err)
		}

		if err := vc.store.PublishOffsetChanged(ctx, offset); err != nil {
			vc.logger.Error("failed to publish clock offset change", "error", err)
		}
	}

	vc.notifyListeners(oldTime, newTime)
	return nil
}

func (vc *VirtualClock) SetTime(t time.Time) error { // ДОБАВЛЕНО: возвращаем error
	vc.mu.Lock()
	realNow := time.Now()
	oldTime := realNow.Add(vc.offset)

	// ИСПРАВЛЕНО: Убран скрытый вызов time.Now() внутри time.Until
	vc.offset = t.Sub(realNow)
	newTime := t
	offset := vc.offset
	vc.mu.Unlock()

	if vc.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// ИСПРАВЛЕНО: Ошибки больше не подавляются, а прерывают выполнение
		if err := vc.store.SaveOffset(ctx, offset); err != nil {
			vc.logger.Error("failed to save clock offset to Redis", "error", err)
			return fmt.Errorf("failed to persist clock offset: %w", err)
		}

		if err := vc.store.PublishOffsetChanged(ctx, offset); err != nil {
			vc.logger.Error("failed to publish clock offset change", "error", err)
			return fmt.Errorf("failed to sync clock across instances: %w", err)
		}
	}

	vc.notifyListeners(oldTime, newTime)
	return nil
}

func (vc *VirtualClock) Offset() time.Duration {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return vc.offset
}

func (vc *VirtualClock) Reset() {
	vc.mu.Lock()
	realNow := time.Now()
	oldTime := realNow.Add(vc.offset)
	vc.offset = 0
	newTime := realNow
	vc.mu.Unlock()

	if vc.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := vc.store.SaveOffset(ctx, 0); err != nil {
			vc.logger.Error("failed to reset clock offset in Redis", "error", err)
		}

		if err := vc.store.PublishOffsetChanged(ctx, 0); err != nil {
			vc.logger.Error("failed to publish clock offset change", "error", err)
		}
	}

	vc.notifyListeners(oldTime, newTime)
}

func (vc *VirtualClock) IsAhead() bool {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return vc.offset > 0
}

func (vc *VirtualClock) notifyListeners(oldTime, newTime time.Time) {
	vc.mu.RLock()
	listeners := make([]TimeJumpListener, len(vc.listeners))
	copy(listeners, vc.listeners)
	vc.mu.RUnlock()

	for _, listener := range listeners {
		listener(oldTime, newTime)
	}
}
