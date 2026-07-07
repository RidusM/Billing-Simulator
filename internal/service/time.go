package service

import (
	"bill-stripe-sim/internal/clock"
	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type (
	RenewalProcessor interface {
		RenewSubscription(ctx context.Context, subID uuid.UUID) (*entity.Invoice, error) // ← Новое имя
	}

	SubscriptionProvider interface {
		GetActiveForRenewal(ctx context.Context, currentTime time.Time) ([]*entity.Subscription, error)
	}

	LockManager interface {
		Lock(ctx context.Context, key string, ttl time.Duration) (func(), error)
	}
)

type TimeService struct {
	clock       *clock.VirtualClock
	processor   RenewalProcessor
	subs        SubscriptionProvider
	cache       LockManager
	log         logger.Logger
	workerCount int
}

func NewTimeService(
	cl *clock.VirtualClock,
	processor RenewalProcessor,
	subs SubscriptionProvider,
	cache LockManager,
	log logger.Logger,
) *TimeService {
	return &TimeService{
		clock:       cl,
		processor:   processor,
		subs:        subs,
		cache:       cache,
		log:         log,
		workerCount: 10,
	}
}

func (s *TimeService) GetCurrentTime() time.Time {
	return s.clock.Now()
}

func (s *TimeService) AdvanceTime(ctx context.Context, d time.Duration) error {
	const op = "service.time.AdvanceTime"

	if err := s.clock.Advance(d); err != nil {
		return fmt.Errorf("%s: advance clock: %w", op, err)
	}

	s.log.LogAttrs(ctx, logger.InfoLevel, "time advanced",
		logger.String("duration", d.String()),
		logger.String("new_time", s.clock.Now().Format(time.RFC3339)),
	)

	return s.CheckAndRenewSubscriptions(ctx)
}

func (s *TimeService) CheckAndRenewSubscriptions(ctx context.Context) error {
	const op = "service.time.CheckAndRenewSubscriptions"
	now := s.clock.Now()

	subs, err := s.subs.GetActiveForRenewal(ctx, now)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if len(subs) == 0 {
		return nil
	}

	s.log.LogAttrs(ctx, logger.InfoLevel, "starting batch renewal",
		logger.Int("count", len(subs)),
	)

	jobs := make(chan *entity.Subscription, len(subs))
	for _, sub := range subs {
		jobs <- sub
	}
	close(jobs)

	var wg sync.WaitGroup
	for i := 0; i < s.workerCount; i++ {
		wg.Go(func() {
			for sub := range jobs {
				s.processSingleSubscription(ctx, sub)
			}
		})
	}

	wg.Wait()
	return nil
}

func (s *TimeService) processSingleSubscription(ctx context.Context, sub *entity.Subscription) {
	unlock, err := s.cache.Lock(ctx, fmt.Sprintf("renewal:%s", sub.ID), time.Minute)
	if err != nil || unlock == nil {
		return
	}
	defer unlock()

	if _, err := s.processor.RenewSubscription(ctx, sub.ID); err != nil { // ← Новое имя
		s.log.LogAttrs(ctx, logger.ErrorLevel, "failed to renew subscription",
			logger.String("sub_id", sub.ID.String()),
			logger.String("error", err.Error()),
		)
	}
}
