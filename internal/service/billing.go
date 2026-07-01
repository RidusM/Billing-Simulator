package service

import (
	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"
	"context"
	"time"

	"github.com/google/uuid"
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
		UpdateStatus(ctx context.Context, id uuid.UUID, status entity.SubscriptionStatus) error
		UpdateNextBilling(ctx context.Context, id uuid.UUID, nextBilling time.Time, periodStart time.Time, periodEnd time.Time) error
		GetActiveForRenewal(ctx context.Context, currentTime time.Time) ([]*entity.Subscription, error)
	}

	CacheRepository interface {
		Set(ctx context.Context, key string, value any, ttl time.Duration) error
		Get(ctx context.Context, key string, dest any) error
		Delete(ctx context.Context, key string) error
		Lock(ctx context.Context, key string, ttl time.Duration) (func(), error)
	}

	TransactionManager interface {
		ExecuteInTransaction(ctx context.Context, txName string, fn func(ctx context.Context) error) error
	}
)

type BillingService struct {
	customers     CustomerRepository
	invoices      InvoiceRepository
	subscriptions SubscriptionRepository
	cache         CacheRepository
	tm            TransactionManager
	log           logger.Logger
}

func NewBillingService(
	customers CustomerRepository,
	invoices InvoiceRepository,
	subscriptions SubscriptionRepository,
	cache CacheRepository,
	tm TransactionManager,
	log logger.Logger,
) *BillingService {
	return &BillingService{
		customers:     customers,
		invoices:      invoices,
		subscriptions: subscriptions,
		cache:         cache,
		tm:            tm,
		log:           log,
	}
}

func (bs *BillingService) CreateCustomer(ctx context.Context, email string) (*entity.Customer, error) {
	return nil, nil
}

func (bs *BillingService) CreateSubscription(ctx context.Context, customerID uuid.UUID, priceID string) (*entity.Subscription, error) {
	return nil, nil
}

func (bs *BillingService) CancelSubscription(ctx context.Context, subID string) error {
	return nil
}

func (bs *BillingService) ProcessRenewal(ctx context.Context, subID uuid.UUID) error {
	return nil
}
