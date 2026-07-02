package service

import (
	"bill-stripe-sim/internal/clock"
	"bill-stripe-sim/pkg/logger"
)

type Services struct {
	Billing      *BillingService
	Time         *TimeService
	Notification *NotificationService
}

type Deps struct {
	Customers     CustomerRepository
	Invoices      InvoiceRepository
	Subscriptions SubscriptionRepository
	Cache         CacheRepository
	TM            TransactionManager
	Log           logger.Logger
	Clock         *clock.VirtualClock
	Sender        EventSender
}

func NewServices(deps Deps) *Services {
	ns := NewNotificationService(deps.Sender)

	bs := NewBillingService(
		deps.Customers,
		deps.Invoices,
		deps.Subscriptions,
		deps.Cache,
		deps.TM,
		deps.Log,
		deps.Clock,
		ns,
	)

	ts := NewTimeService(
		deps.Clock,
		bs,
		deps.Subscriptions,
		deps.Cache,
		deps.Log,
	)

	return &Services{
		Billing:      bs,
		Time:         ts,
		Notification: ns,
	}
}
