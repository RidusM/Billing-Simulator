package entity

import "errors"

var (
	ErrDataNotFound    = errors.New("data not found")
	ErrConflictingData = errors.New("conflicting data")
	ErrInvalidData     = errors.New("invalid data")
	ErrInternal        = errors.New("internal server error")

	ErrCustomerNotFound = errors.New("customer not found")

	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrSubscriptionAlreadyCanceled = errors.New("subscription is already canceled")
	ErrSubscriptionNotActive       = errors.New("subscription is not active")
	ErrInvalidPrice         = errors.New("invalid price id")

	ErrInvoiceNotFound    = errors.New("invoice not found")
	ErrInvoiceAlreadyPaid = errors.New("invoice is already paid")
	ErrInvoiceNotPayable  = errors.New("invoice is not in a payable state")
)
