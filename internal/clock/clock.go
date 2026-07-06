package clock

import (
	"bill-stripe-sim/pkg/logger"
	"context"
	"fmt"
	"sync"
	"time"
)

type Store interface {
	SaveOffset(ctx context.Context, offset time.Duration) error
	LoadOffset(ctx context.Context) (time.Duration, error)
}

type TimeJumpListener func(oldTime, newTime time.Time)

type VirtualClock struct {
	mu     sync.RWMutex
	offset time.Duration
	store  Store

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

func (vc *VirtualClock) Advance(d time.Duration) {
	vc.mu.Lock()

	realNow := time.Now()
	oldTime := realNow.Add(vc.offset)

	vc.offset += d

	newTime := realNow.Add(vc.offset)

	listeners := vc.listeners
	vc.mu.Unlock()

	for _, listener := range listeners {
		go listener(oldTime, newTime)
	}
}

func (vc *VirtualClock) SetTime(t time.Time) {
	vc.mu.Lock()
	vc.offset = time.Until(t)
	offset := vc.offset
	vc.mu.Unlock()

	if vc.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := vc.store.SaveOffset(ctx, offset); err != nil {
			vc.logger.Error("failed to save clock offset to Redis")
		}
	}
}

func (vc *VirtualClock) Offset() time.Duration {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return vc.offset
}

func (vc *VirtualClock) Reset() {
	vc.mu.Lock()
	vc.offset = 0
	vc.mu.Unlock()

	if vc.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := vc.store.SaveOffset(ctx, 0); err != nil {
			vc.logger.Error("failed to reset clock offset in Redis",
				"error", err,
			)
		}
	}
}

func (vc *VirtualClock) IsAhead() bool {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return vc.offset > 0
}

func (vc *VirtualClock) notifyListeners(oldTime, newTime time.Time) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	for _, listener := range vc.listeners {
		go listener(oldTime, newTime)
	}
}
