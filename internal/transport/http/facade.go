package handler

import (
	"context"
	"fmt"
	"time"

	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/internal/service"
)

// BillingFacade склеивает 4 независимых сервиса (Customer/Billing/Time/PaymentRate)
// в один интерфейс handler.BillingService, который ожидает HTTP-слой.
//
// Почему facade, а не один "God Service": домен специально разбит на CustomerService,
// BillingService, TimeService, PaymentRateManager — каждый с одной ответственностью и
// своим набором репозиториев/зависимостей. Facade — единственное место, которое знает
// про все четыре сразу; сами сервисы друг про друга ничего не знают.
//
// ВАЖНО: интерфейс handler.BillingService (billing_transport.go) нужно поправить —
// customer_id и subscription_id всегда public_id string, никогда uuid.UUID
// (аналогично тому, как уже сделано для price_id):
//
//	CreateSubscription(ctx context.Context, customerID string, priceID string) (*entity.Subscription, error)
//	GetSubscription(ctx context.Context, subID string) (*entity.Subscription, error)
//
// И 2 небольшие правки в service-пакете (тонкие read-методы, которых сейчас не хватает):
//
//	// service/billing.go
//	func (bs *BillingService) GetSubscriptionByPublicID(ctx context.Context, publicID string) (*entity.Subscription, error) {
//	    return bs.subscriptions.GetByPublicID(ctx, publicID)
//	}
//
//	// service/price.go
//	func (s *PriceService) GetPriceByPublicID(ctx context.Context, publicID string) (*entity.Price, error) {
//	    return s.prices.GetByPublicID(ctx, publicID)
//	}
type BillingFacade struct {
	customers *service.CustomerService
	billing   *service.BillingService
	time      *service.TimeService
	rates     *service.PaymentRateManager
	prices    *service.PriceService
}

func NewBillingFacade(
	customers *service.CustomerService,
	billing *service.BillingService,
	timeSvc *service.TimeService,
	rates *service.PaymentRateManager,
	prices *service.PriceService,
) *BillingFacade {
	return &BillingFacade{
		customers: customers,
		billing:   billing,
		time:      timeSvc,
		rates:     rates,
		prices:    prices,
	}
}

func (f *BillingFacade) CreateCustomer(ctx context.Context, email string) (*entity.Customer, error) {
	return f.customers.CreateCustomer(ctx, email)
}

// CreateSubscription принимает customer_id и price_id как ПУБЛИЧНЫЕ id ("cus_xxx"/"price_xxx"),
// не как uuid — резолвим оба во внутренние uuid перед вызовом BillingService.
func (f *BillingFacade) CreateSubscription(ctx context.Context, customerID, priceID string) (*entity.Subscription, error) {
	const op = "handler.BillingFacade.CreateSubscription"

	customer, err := f.customers.GetCustomer(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("%s: resolve customer: %w", op, err)
	}

	price, err := f.prices.GetPriceByPublicID(ctx, priceID)
	if err != nil {
		return nil, fmt.Errorf("%s: resolve price: %w", op, err)
	}

	return f.billing.CreateSubscription(ctx, customer.ID, price.ID)
}

func (f *BillingFacade) CancelSubscription(ctx context.Context, subID string, atPeriodEnd bool) error {
	return f.billing.CancelSubscription(ctx, subID, atPeriodEnd)
}

func (f *BillingFacade) GetSubscription(ctx context.Context, subID string) (*entity.Subscription, error) {
	return f.billing.GetSubscriptionByPublicID(ctx, subID)
}

func (f *BillingFacade) AdvanceTime(ctx context.Context, d time.Duration) error {
	return f.time.AdvanceTime(ctx, d)
}

func (f *BillingFacade) GetCurrentTime() time.Time {
	return f.time.GetCurrentTime()
}

func (f *BillingFacade) GetPaymentSuccessRate() float64 {
	return f.rates.GetSuccessRate()
}

func (f *BillingFacade) SetPaymentSuccessRate(rate float64) error {
	return f.rates.SetSuccessRate(rate)
}
