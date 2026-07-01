package clock

import (
	"context"
	"sync"
	"time"
)

type Store interface {
	SaveOffset(ctx context.Context, offset time.Duration) error
	LoadOffset(ctx context.Context) (time.Duration, error)
}

type VirtualClock struct {
	mu     sync.RWMutex
	offset time.Duration
	store  Store
}

func NewVirtualClock(s Store) *VirtualClock {
	vc := &VirtualClock{
		store: s,
	}

	if s != nil {
		if offset, err := s.LoadOffset(context.Background()); err == nil {
			vc.offset = offset
		}
	}

	return vc
}

func (vc *VirtualClock) Now() time.Time {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return time.Now().Add(vc.offset)
}

func (vc *VirtualClock) Advance(d time.Duration) {
	vc.mu.Lock()
	vc.offset += d
	offset := vc.offset
	vc.mu.Unlock()

	if vc.store != nil {
		_ = vc.store.SaveOffset(context.Background(), offset)
	}
}

func (vc *VirtualClock) SetTime(t time.Time) {
	vc.mu.Lock()
	vc.offset = time.Until(t)
	offset := vc.offset
	vc.mu.Unlock()

	if vc.store != nil {
		_ = vc.store.SaveOffset(context.Background(), offset)
	}
}
