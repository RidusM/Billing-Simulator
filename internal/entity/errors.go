package entity

import "errors"

var (
	ErrDataNotFound    = errors.New("data not found")
	ErrConflictingData = errors.New("conflicting data")
	ErrInvalidData     = errors.New("invalid data")
	ErrInternal        = errors.New("internal server error")

	ErrCustomerNotFound = errors.New("customer not found")

	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrSubscriptionCanceled = errors.New("subscription is already canceled")
	ErrInvalidPrice         = errors.New("invalid price id")

	ErrInvoiceNotFound    = errors.New("invoice not found")
	ErrInvoiceAlreadyPaid = errors.New("invoice is already paid")
)
