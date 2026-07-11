package entity

import "errors"

var (
	ErrDataNotFound    = errors.New("data not found")
	ErrConflictingData = errors.New("conflicting data")
	ErrInvalidData     = errors.New("invalid data")
	ErrInternal        = errors.New("internal server error")

	ErrCustomerNotFound = errors.New("customer not found")

	ErrProductNotFound = errors.New("product not found")
	ErrPriceNotFound   = errors.New("price not found")
	ErrInvalidPrice    = errors.New("invalid price id")

	ErrSubscriptionNotFound        = errors.New("subscription not found")
	ErrSubscriptionAlreadyCanceled = errors.New("subscription is already canceled")
	ErrSubscriptionNotActive       = errors.New("subscription is not active")
	ErrSubscriptionUnpaid          = errors.New("subscription unpaid")

	ErrInvoiceNotFound    = errors.New("invoice not found")
	ErrInvoiceAlreadyPaid = errors.New("invoice is already paid")
	ErrInvoiceNotPayable  = errors.New("invoice is not in a payable state")

	ErrPaymentIntentNotFound = errors.New("payment intent not found")
	ErrInvalidCaptureAmount  = errors.New("payment invalid capture amount")

	ErrWebhookEndpointNotFound = errors.New("webhook endpoint not found")
	ErrWebhookEndpointDisabled = errors.New("webhook endpoint is disabled")
)
