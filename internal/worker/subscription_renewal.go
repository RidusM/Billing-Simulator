package worker

import (
	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"
	"context"
	"time"

	"github.com/google/uuid"
)

type SubscriptionRepository interface {
	GetActiveForRenewal(ctx context.Context, currentTime time.Time) ([]*entity.Subscription, error)
}

type RenewalProcessor interface {
	RenewSubscription(ctx context.Context, subID uuid.UUID) (*entity.Invoice, error)
}

type SubscriptionRenewalWorker struct {
	subs      SubscriptionRepository
	processor RenewalProcessor
	clock     VirtualClock
	log       logger.Logger
	cfg       SubscriptionRenewalConfig
	done      chan struct{}
}

type SubscriptionRenewalConfig struct {
	PollInterval time.Duration
	BatchSize    int
}

func DefaultSubscriptionRenewalConfig() SubscriptionRenewalConfig {
	return SubscriptionRenewalConfig{
		PollInterval: 1 * time.Minute, // Проверяем каждую минуту
		BatchSize:    100,
	}
}

func NewSubscriptionRenewalWorker(
	subs SubscriptionRepository,
	processor RenewalProcessor,
	clock VirtualClock,
	log logger.Logger,
	cfg SubscriptionRenewalConfig,
) *SubscriptionRenewalWorker {
	return &SubscriptionRenewalWorker{
		subs:      subs,
		processor: processor,
		clock:     clock,
		log:       log,
		cfg:       cfg,
		done:      make(chan struct{}),
	}
}

func (w *SubscriptionRenewalWorker) Start(ctx context.Context) {
	w.log.Info("starting subscription renewal worker",
		"poll_interval", w.cfg.PollInterval,
	)

	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("subscription renewal worker stopped")
			return
		case <-w.done:
			return
		case <-ticker.C:
			w.processRenewals(ctx)
		}
	}
}

func (w *SubscriptionRenewalWorker) Stop() {
	close(w.done)
}

func (w *SubscriptionRenewalWorker) processRenewals(ctx context.Context) {
	now := w.clock.Now()
	subs, err := w.subs.GetActiveForRenewal(ctx, now)
	if err != nil {
		w.log.Error("failed to get subscriptions for renewal", "error", err)
		return
	}

	if len(subs) == 0 {
		return
	}

	w.log.Info("processing subscription renewals", "count", len(subs))

	for _, sub := range subs {
		if _, err := w.processor.RenewSubscription(ctx, sub.ID); err != nil {
			w.log.Error("failed to renew subscription",
				"subscription_id", sub.ID,
				"error", err,
			)
		} else {
			w.log.Info("subscription renewed successfully",
				"subscription_id", sub.ID,
			)
		}
	}
}
