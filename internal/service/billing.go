package service

import (
	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type (
	OutboxPoller interface {
		GetUnprocessed(ctx context.Context, limit int, olderThan time.Duration) ([]*entity.OutboxEvent, error)
		MarkProcessed(ctx context.Context, id uuid.UUID) error
		MarkFailed(ctx context.Context, id uuid.UUID, errorMsg string) error
		DeleteOldProcessed(ctx context.Context, olderThan time.Duration) (int64, error)
	}
	SubscriptionRepository interface {
		Create(ctx context.Context, s *entity.Subscription) error
		GetByID(ctx context.Context, id uuid.UUID) (*entity.Subscription, error)
		GetByPublicID(ctx context.Context, publicID string) (*entity.Subscription, error)
		Update(ctx context.Context, s *entity.Subscription) error
	}

	InvoiceRepository interface {
		Create(ctx context.Context, i *entity.Invoice) error
		GetByID(ctx context.Context, id uuid.UUID) (*entity.Invoice, error) // ← ДОБАВИТЬ
		GetByPublicID(ctx context.Context, publicID string) (*entity.Invoice, error)
		Update(ctx context.Context, i *entity.Invoice) error // ← ДОБАВИТЬ
	}

	PriceReader interface {
		GetByID(ctx context.Context, id uuid.UUID) (*entity.Price, error)
	}

	TransactionManager interface {
		ExecuteInTransaction(ctx context.Context, txName string, fn func(ctx context.Context) error) error
	}
)

type BillingService struct {
	subscriptions SubscriptionRepository
	invoices      InvoiceRepository
	price         PriceReader
	dispatcher    *EventDispatcher
	tm            TransactionManager
	log           logger.Logger
	clock         VirtualClock
	rateManager   *PaymentRateManager
}

func NewBillingService(
	subscriptions SubscriptionRepository,
	invoices InvoiceRepository,
	price PriceRepository,
	dispatcher *EventDispatcher,
	tm TransactionManager,
	log logger.Logger,
	clock VirtualClock,
	rateManager *PaymentRateManager,
) *BillingService {
	return &BillingService{
		subscriptions: subscriptions,
		invoices:      invoices,
		price:         price,
		dispatcher:    dispatcher,
		tm:            tm,
		log:           log,
		clock:         clock,
		rateManager:   rateManager,
	}
}

func (bs *BillingService) CreateSubscription(ctx context.Context, customerID uuid.UUID, priceID uuid.UUID) (*entity.Subscription, error) {
	const op = "service.billing.CreateSubscription"
	var sub *entity.Subscription
	var inv *entity.Invoice
	err := bs.tm.ExecuteInTransaction(ctx, op, func(ctx context.Context) error {
		price, err := bs.price.GetByID(ctx, priceID)
		if err != nil {
			return fmt.Errorf("get price: %w", err)
		}
		now := bs.clock.Now()
		nextBilling := price.NextBillingDate(now)
		sub = entity.NewSubscription(customerID, price.ID, now, nextBilling, now)
		if err := bs.subscriptions.Create(ctx, sub); err != nil {
			return fmt.Errorf("create subscription: %w", err)
		}

		inv = entity.NewInvoice(
			customerID,
			sub.CustomerPublicID,
			&sub.ID,
			&sub.PublicID,
			price.Amount,
			price.Currency,
			now,
		)

		if err := inv.MarkPaid(now); err != nil {
			return fmt.Errorf("mark initial invoice paid: %w", err)
		}
		if err := bs.invoices.Create(ctx, inv); err != nil {
			return fmt.Errorf("create initial invoice: %w", err)
		}
		events := sub.GetAndClearEvents()
		events = append(events, inv.GetAndClearEvents()...)
		if events.HasEvents() {
			if err := bs.dispatcher.Dispatch(ctx, events); err != nil {
				return fmt.Errorf("dispatch events: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return sub, nil
}

func (bs *BillingService) CancelSubscription(ctx context.Context, subID string, atPeriodEnd bool) error {
	const op = "service.billing.CancelSubscription"

	var sub *entity.Subscription

	err := bs.tm.ExecuteInTransaction(ctx, op, func(ctx context.Context) error {
		var err error
		sub, err = bs.subscriptions.GetByPublicID(ctx, subID)
		if err != nil {
			return fmt.Errorf("get subscription: %w", err)
		}

		now := bs.clock.Now()
		if err := sub.Cancel(now, atPeriodEnd); err != nil {
			return fmt.Errorf("cancel subscription: %w", err)
		}

		if err := bs.subscriptions.Update(ctx, sub); err != nil {
			return fmt.Errorf("update subscription: %w", err)
		}

		events := sub.GetAndClearEvents()
		if events.HasEvents() {
			if err := bs.dispatcher.Dispatch(ctx, events); err != nil {
				return fmt.Errorf("dispatch events: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (bs *BillingService) RenewSubscription(ctx context.Context, subID uuid.UUID) (*entity.Invoice, error) {
	const op = "service.billing.RenewSubscription"
	var sub *entity.Subscription
	var inv *entity.Invoice
	err := bs.tm.ExecuteInTransaction(ctx, op, func(ctx context.Context) error {
		var err error
		sub, err = bs.subscriptions.GetByID(ctx, subID)
		if err != nil {
			return fmt.Errorf("get subscription: %w", err)
		}
		price, err := bs.price.GetByID(ctx, sub.PriceID)
		if err != nil {
			return fmt.Errorf("get price: %w", err)
		}
		now := bs.clock.Now()
		if sub.NextBillingAt.After(now) {
			return nil
		}
		inv, err = sub.Renew(now, price)
		if err != nil {
			return fmt.Errorf("renew subscription: %w", err)
		}

		simStatus := simulatePayment(ctx, bs.rateManager)

		if simStatus == entity.InvoiceStatusPaid {
			if err := inv.MarkPaid(now); err != nil {
				return fmt.Errorf("mark paid: %w", err)
			}
			sub.MarkPaid(now)
		} else {
			inv.AttemptCount++
			isFinalAttempt := inv.AttemptCount >= 3

			if isFinalAttempt {
				sub.MarkCanceled(now)
			} else {
				sub.MarkPastDue(now)
			}

			if err := inv.MarkPaymentFailed(now, "card_declined", isFinalAttempt); err != nil {
				return fmt.Errorf("mark payment failed: %w", err)
			}
		}

		if err := bs.invoices.Create(ctx, inv); err != nil {
			return fmt.Errorf("create renewal invoice: %w", err)
		}
		if err := bs.subscriptions.Update(ctx, sub); err != nil {
			return fmt.Errorf("update subscription: %w", err)
		}
		events := sub.GetAndClearEvents()
		events = append(events, inv.GetAndClearEvents()...)
		if events.HasEvents() {
			if err := bs.dispatcher.Dispatch(ctx, events); err != nil {
				return fmt.Errorf("dispatch events: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return inv, nil
}
