package service

import (
	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

type (
	CustomerRepository interface {
		Create(ctx context.Context, c *entity.Customer) error
		GetByID(ctx context.Context, id uuid.UUID) (*entity.Customer, error)
		GetByPublicID(ctx context.Context, publicID string) (*entity.Customer, error)
		Delete(ctx context.Context, id uuid.UUID) error
	}

	InvoiceRepository interface {
		Create(ctx context.Context, i *entity.Invoice) error
		GetByID(ctx context.Context, id uuid.UUID) (*entity.Invoice, error)
		GetByPublicID(ctx context.Context, publicID string) (*entity.Invoice, error)
		UpdateStatus(ctx context.Context, id uuid.UUID, status entity.InvoiceStatus) error
		IncrementAttempt(ctx context.Context, id uuid.UUID) error
		GetByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*entity.Invoice, error)
	}

	SubscriptionRepository interface {
		Create(ctx context.Context, s *entity.Subscription) error
		GetByID(ctx context.Context, id uuid.UUID) (*entity.Subscription, error)
		GetByPublicID(ctx context.Context, publicID string) (*entity.Subscription, error)
		GetAllActive(ctx context.Context) ([]*entity.Subscription, error)
		UpdateStatus(ctx context.Context, id uuid.UUID, status entity.SubscriptionStatus) error
		UpdateNextBilling(ctx context.Context, id uuid.UUID, nextBilling time.Time, periodStart time.Time, periodEnd time.Time) error
		GetActiveForRenewal(ctx context.Context, currentTime time.Time) ([]*entity.Subscription, error)
	}

	CacheRepository interface {
		Set(ctx context.Context, key string, value any, ttl time.Duration) error
		SetBatch(ctx context.Context, items map[string]any, ttl time.Duration) error
		Get(ctx context.Context, key string, dest any) error
		Delete(ctx context.Context, key string) error
		Lock(ctx context.Context, key string, ttl time.Duration) (func(), error)
	}

	TransactionManager interface {
		ExecuteInTransaction(ctx context.Context, txName string, fn func(ctx context.Context) error) error
	}

	TimeProvider interface {
		Now() time.Time
	}
)

type BillingService struct {
	customers     CustomerRepository
	invoices      InvoiceRepository
	subscriptions SubscriptionRepository
	cache         CacheRepository
	clock         TimeProvider
	tm            TransactionManager
	log           logger.Logger
	notification  *NotificationService
	sf            singleflight.Group
}

func NewBillingService(
	customers CustomerRepository,
	invoices InvoiceRepository,
	subscriptions SubscriptionRepository,
	cache CacheRepository,
	tm TransactionManager,
	log logger.Logger,
	clock TimeProvider,
	notification *NotificationService,
) *BillingService {
	return &BillingService{
		customers:     customers,
		invoices:      invoices,
		subscriptions: subscriptions,
		cache:         cache,
		tm:            tm,
		log:           log,
		clock:         clock,
		notification:  notification,
	}
}

func (bs *BillingService) CreateCustomer(ctx context.Context, email string) (*entity.Customer, error) {
	const op = "service.billing.CreateCustomer"
	log := bs.log.With("op", op)
	start := time.Now()
	defer logSlowOperation(ctx, log, op, start)

	log.LogAttrs(ctx, logger.InfoLevel, "starting customer creation",
		logger.String("email", email),
	)

	var c *entity.Customer
	err := bs.tm.ExecuteInTransaction(ctx, op, func(ctx context.Context) error {
		c = &entity.Customer{
			ID:        uuid.New(),
			PublicID:  generatePublicID("cus"),
			Email:     email,
			CreatedAt: bs.clock.Now(),
		}

		if err := bs.customers.Create(ctx, c); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.LogAttrs(ctx, logger.InfoLevel, "customer created",
		logger.String("customer_id", c.ID.String()),
		logger.String("public_id", c.PublicID),
		logger.Duration("duration", time.Since(start)),
	)

	return c, nil
}

func (bs *BillingService) CreateSubscription(ctx context.Context, customerID uuid.UUID, priceID string) (*entity.Subscription, error) {
	const op = "service.billing.CreateSubscription"
	log := bs.log.With("op", op)
	start := time.Now()
	defer logSlowOperation(ctx, log, op, start)

	log.LogAttrs(ctx, logger.InfoLevel, "starting subscription creation",
		logger.String("customer_id", customerID.String()),
		logger.String("price_id", priceID),
	)

	var sub *entity.Subscription
	var inv *entity.Invoice

	err := bs.tm.ExecuteInTransaction(ctx, op, func(ctx context.Context) error {
		_, err := bs.customers.GetByID(ctx, customerID)
		if err != nil {
			return fmt.Errorf("check customer: %w", err)
		}

		now := bs.clock.Now()
		_, nextBilling := calculateNextPeriod(now)

		sub = &entity.Subscription{
			ID:                 uuid.New(),
			PublicID:           generatePublicID("sub"),
			CustomerID:         customerID,
			Status:             entity.SubscriptionStatusActive,
			PriceID:            priceID,
			CurrentPeriodStart: now,
			CurrentPeriodEnd:   nextBilling,
			NextBillingAt:      nextBilling,
			CreatedAt:          now,
		}

		if err := bs.subscriptions.Create(ctx, sub); err != nil {
			return fmt.Errorf("create subscription: %w", err)
		}

		inv = &entity.Invoice{
			ID:             uuid.New(),
			PublicID:       generatePublicID("in"),
			SubscriptionID: &sub.ID,
			CustomerID:     customerID,
			Amount:         getRandomAmount(),
			Currency:       getRandomCurrency(),
			Status:         entity.InvoiceStatusPaid,
			CreatedAt:      now,
		}

		if err := bs.invoices.Create(ctx, inv); err != nil {
			return fmt.Errorf("create initial invoice: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	_ = bs.notification.NotifySubscriptionUpdated(ctx, sub)
	_ = bs.notification.NotifyInvoiceCreated(ctx, inv)

	log.LogAttrs(ctx, logger.InfoLevel, "subscription created",
		logger.String("sub_id", sub.PublicID),
		logger.String("inv_id", inv.PublicID),
		logger.Duration("duration", time.Since(start)),
	)

	return sub, nil
}

func (bs *BillingService) CancelSubscription(ctx context.Context, subID string) error {
	const op = "service.billing.CancelSubscription"
	log := bs.log.With("op", op)
	start := time.Now()
	defer logSlowOperation(ctx, log, op, start)

	log.LogAttrs(ctx, logger.InfoLevel, "start cancel subscription",
		logger.String("sub_id", subID),
	)

	err := bs.tm.ExecuteInTransaction(ctx, op, func(ctx context.Context) error {
		sub, err := bs.subscriptions.GetByPublicID(ctx, subID)
		if err != nil {
			return fmt.Errorf("get subscription: %w", err)
		}

		if sub.Status != entity.SubscriptionStatusActive && sub.Status != entity.SubscriptionStatusPastDue {
			return fmt.Errorf("subscription is not active for cancel: %s", sub.Status)
		}

		return bs.subscriptions.UpdateStatus(ctx, sub.ID, entity.SubscriptionStatusCanceled)
	})

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_ = bs.cache.Delete(ctx, fmt.Sprintf("sub:%s", subID))

	return nil
}

func (bs *BillingService) ProcessRenewal(ctx context.Context, subID uuid.UUID) error {
	const op = "service.billing.ProcessRenewal"
	log := bs.log.With("op", op, "sub_id", subID.String())
	start := time.Now()
	defer logSlowOperation(ctx, log, op, start)

	var sub *entity.Subscription
	var inv *entity.Invoice

	err := bs.tm.ExecuteInTransaction(ctx, op, func(ctx context.Context) error {
		var err error
		sub, err = bs.subscriptions.GetByID(ctx, subID)
		if err != nil {
			return fmt.Errorf("get subscription: %w", err)
		}

		if sub.Status != entity.SubscriptionStatusActive && sub.Status != entity.SubscriptionStatusPastDue {
			return fmt.Errorf("subscription is not active for renewal: %s", sub.Status)
		}

		if sub.NextBillingAt.After(bs.clock.Now()) {
			return nil
		}

		now := bs.clock.Now()
		inv = &entity.Invoice{
			ID:             uuid.New(),
			PublicID:       generatePublicID("in"),
			SubscriptionID: &sub.ID,
			CustomerID:     sub.CustomerID,
			Amount:         getRandomAmount(),
			Currency:       getRandomCurrency(),
			Status:         simulatePayment(),
			CreatedAt:      now,
		}

		if err := bs.invoices.Create(ctx, inv); err != nil {
			return fmt.Errorf("create renewal invoice: %w", err)
		}

		if inv.Status == entity.InvoiceStatusPaid {
			_, nextBilling := calculateNextPeriod(sub.CurrentPeriodEnd)
			err = bs.subscriptions.UpdateNextBilling(ctx, sub.ID, nextBilling, sub.CurrentPeriodEnd, nextBilling)
			if err != nil {
				return fmt.Errorf("update subscription dates: %w", err)
			}
			_ = bs.subscriptions.UpdateStatus(ctx, sub.ID, entity.SubscriptionStatusActive)
		} else {
			_ = bs.subscriptions.UpdateStatus(ctx, sub.ID, entity.SubscriptionStatusPastDue)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if inv != nil {
		_ = bs.cache.Delete(ctx, fmt.Sprintf("sub:%s", subID))
		_ = bs.notification.NotifySubscriptionUpdated(ctx, sub)
		_ = bs.notification.NotifyInvoiceCreated(ctx, inv)
	}

	log.LogAttrs(ctx, logger.InfoLevel, "subscription renewal processed",
		logger.Duration("duration", time.Since(start)),
	)

	return nil
}

func (bs *BillingService) GetSubscription(ctx context.Context, subID uuid.UUID) (*entity.Subscription, error) {
	const op = "service.billing.GetSubscription"
	key := fmt.Sprintf("sub:%s", subID)

	var sub entity.Subscription
	err := bs.cache.Get(ctx, key, &sub)
	if err == nil && sub.ID != uuid.Nil {
		return &sub, nil
	}

	v, err, _ := bs.sf.Do(key, func() (interface{}, error) {
		s, err := bs.subscriptions.GetByID(ctx, subID)
		if err != nil {
			return nil, err
		}

		_ = bs.cache.Set(ctx, key, s, time.Hour)
		return s, nil
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return v.(*entity.Subscription), nil
}

func (bs *BillingService) RestoreCache(ctx context.Context) error {
	const op = "service.billing.RestoreCache"
	log := bs.log.With("op", op)
	start := time.Now()
	defer logSlowOperation(ctx, log, op, start)

	subs, err := bs.subscriptions.GetAllActive(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	items := make(map[string]any, len(subs))
	for _, sub := range subs {
		key := fmt.Sprintf("sub:%s", sub.ID)
		items[key] = sub
	}

	if len(items) > 0 {
		if err := bs.cache.SetBatch(ctx, items, time.Hour); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	log.LogAttrs(ctx, logger.InfoLevel, "cache restored",
		logger.Int("count", len(subs)),
		logger.Duration("duration", time.Since(start)),
	)

	return nil
}
