package service

import (
	"bill-stripe-sim/internal/clock"
	"context"
	"time"
)

type TimeService struct {
	clock   *clock.VirtualClock
	billing *BillingService
}

func NewTimeService(cl *clock.VirtualClock, billing *BillingService) *TimeService {
	return &TimeService{
		clock:   cl,
		billing: billing,
	}
}

func (s *TimeService) GetCurrentTime() time.Time {
	return s.clock.Now()
}

func (s *TimeService) AdvanceTime(ctx context.Context, d time.Duration) error {
	s.clock.Advance(d)
	return s.CheckAndRenewSubscriptions(ctx)
}

func (s *TimeService) CheckAndRenewSubscriptions(ctx context.Context) error {
	return nil
}
