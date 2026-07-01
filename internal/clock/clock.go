package clock

import (
	"sync"
	"time"
)

type VirtualClock struct {
	mu sync.RWMutex
	offset time.Duration
}

func NewVirtualClock() *VirtualClock {
	return &VirtualClock{}
}

func (vc *VirtualClock) Now() time.Time {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return time.Now().Add(vc.offset)
}

func (vc *VirtualClock) Advance(d time.Duration) {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.offset += d
}

func (vc *VirtualClock) SetTime(t time.Time) {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.offset = time.Until(t)
}