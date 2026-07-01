package service

type Services struct {
	Billing      *BillingService
	Time         *TimeService
	Notification *NotificationService
}

func NewServices(
	billing *BillingService,
	timeService *TimeService,
	notification *NotificationService,
) *Services {
	return &Services{
		Billing:      billing,
		Time:         timeService,
		Notification: notification,
	}
}
