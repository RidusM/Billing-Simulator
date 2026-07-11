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

	CustomerRepository interface {
		Create(ctx context.Context, c *entity.Customer) error
		GetByID(ctx context.Context, id uuid.UUID) (*entity.Customer, error)
		GetByPublicID(ctx context.Context, publicID string) (*entity.Customer, error)
		Update(ctx context.Context, c *entity.Customer) error
	}

	TransactionManager interface {
		ExecuteInTransaction(ctx context.Context, txName string, fn func(ctx context.Context) error) error
	}
)

type BillingService struct {
	subscriptions SubscriptionRepository
	invoices      InvoiceRepository
	customers     CustomerRepository // ← ДОБАВИТЬ
	price         PriceRepository
	dispatcher    *EventDispatcher
	tm            TransactionManager
	log           logger.Logger
	clock         VirtualClock
	rateManager   *PaymentRateManager
}

func NewBillingService(
	subscriptions SubscriptionRepository,
	invoices InvoiceRepository,
	customers CustomerRepository, // ← ДОБАВИТЬ
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
		customers:     customers, // ← ДОБАВИТЬ
		price:         price,
		dispatcher:    dispatcher,
		tm:            tm,
		log:           log,
		clock:         clock,
		rateManager:   rateManager,
	}
}

// ✅ ПРАВИЛЬНОЕ:
func (bs *BillingService) CreateSubscription(
	ctx context.Context,
	customerID uuid.UUID,
	priceID uuid.UUID,
) (*entity.Subscription, error) {
	const op = "service.billing.CreateSubscription"
	var sub *entity.Subscription
	var inv *entity.Invoice

	err := bs.tm.ExecuteInTransaction(ctx, op, func(ctx context.Context) error {
		// Получить customer, чтобы достать его PublicID
		customer, err := bs.customers.GetByID(ctx, customerID)
		if err != nil {
			return fmt.Errorf("get customer: %w", err)
		}

		price, err := bs.price.GetByID(ctx, priceID)
		if err != nil {
			return fmt.Errorf("get price: %w", err)
		}

		now := bs.clock.Now()
		nextBilling := price.NextBillingDate(now)

		// Теперь всё определено
		sub, err = entity.NewSubscription(
			customerID,
			price.ID,
			customer.PublicID, // ← Теперь есть!
			price.PublicID,
			now,
			nextBilling,
			now,
		)
		if err != nil {
			return fmt.Errorf("create subscription: %w", err)
		}

		if err := bs.subscriptions.Create(ctx, sub); err != nil {
			return fmt.Errorf("create subscription: %w", err)
		}

		// Создаём инвойс
		inv, err = entity.NewInvoice(
			customerID,
			customer.PublicID, // ← Используем customer.PublicID
			&sub.ID,
			&sub.PublicID,
			price.Amount,
			price.Currency,
			now,
		)
		if err != nil {
			return fmt.Errorf("create invoice: %w", err)
		}

		// MarkPaid корректно оплачивает инвойс полностью
		if err := inv.MarkPaid(now); err != nil {
			return fmt.Errorf("mark invoice paid: %w", err)
		}

		if err := bs.invoices.Create(ctx, inv); err != nil {
			return fmt.Errorf("create invoice: %w", err)
		}

		// События
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
			isFinalAttempt := (inv.AttemptCount + 1) >= 3
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
