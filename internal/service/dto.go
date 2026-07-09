package service

import (
	"time"

	"github.com/google/uuid"
)

// DTOs for serialization — these are transport-layer concerns
type CustomerCreatedEventDTO struct {
	CustomerID    uuid.UUID `json:"customer_id"`
	CustomerPubID string    `json:"customer_public_id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	CreatedAt     time.Time `json:"created_at"`
}

type InvoiceCreatedEventDTO struct {
	InvoiceID         uuid.UUID  `json:"invoice_id"`
	InvoicePubID      string     `json:"invoice_public_id"`
	CustomerID        uuid.UUID  `json:"customer_id"`
	CustomerPubID     string     `json:"customer_public_id"`
	SubscriptionID    *uuid.UUID `json:"subscription_id,omitempty"`
	SubscriptionPubID *string    `json:"subscription_public_id,omitempty"`
	Amount            int64      `json:"amount"`
	Currency          string     `json:"currency"`
	Status            string     `json:"status"`
	CreatedAt         time.Time  `json:"created_at"`
}

type InvoicePaidEventDTO struct {
	InvoiceID      uuid.UUID  `json:"invoice_id"`
	InvoicePubID   string     `json:"invoice_public_id"`
	CustomerID     uuid.UUID  `json:"customer_id"`
	CustomerPubID  string     `json:"customer_public_id"`
	SubscriptionID *uuid.UUID `json:"subscription_id,omitempty"`
	Amount         int64      `json:"amount"`
	Currency       string     `json:"currency"`
	PaidAt         time.Time  `json:"paid_at"`
}

type InvoicePaymentFailedEventDTO struct {
	InvoiceID      uuid.UUID  `json:"invoice_id"`
	InvoicePubID   string     `json:"invoice_public_id"`
	CustomerID     uuid.UUID  `json:"customer_id"`
	CustomerPubID  string     `json:"customer_public_id"`
	SubscriptionID *uuid.UUID `json:"subscription_id,omitempty"`
	Amount         int64      `json:"amount"`
	Currency       string     `json:"currency"`
	ErrorCode      string     `json:"error_code"`
	FailedAt       time.Time  `json:"failed_at"`
}

type SubscriptionCreatedEventDTO struct {
	SubscriptionID    uuid.UUID `json:"subscription_id"`
	SubscriptionPubID string    `json:"subscription_public_id"`
	CustomerID        uuid.UUID `json:"customer_id"`
	CustomerPubID     string    `json:"customer_public_id"`
	PriceID           uuid.UUID `json:"price_id"`
	PricePubID        string    `json:"price_public_id"`
	Status            string    `json:"status"`
	CurrentPeriodEnd  time.Time `json:"current_period_end"`
	NextBillingAt     time.Time `json:"next_billing_at"`
	CreatedAt         time.Time `json:"created_at"`
}

type SubscriptionCanceledEventDTO struct {
	SubscriptionID    uuid.UUID `json:"subscription_id"`
	SubscriptionPubID string    `json:"subscription_public_id"`
	CustomerID        uuid.UUID `json:"customer_id"`
	CustomerPubID     string    `json:"customer_public_id"`
	Status            string    `json:"status"`
	CanceledAt        time.Time `json:"canceled_at"`
	AtPeriodEnd       bool      `json:"at_period_end"`
}

type SubscriptionRenewedEventDTO struct {
	SubscriptionID    uuid.UUID `json:"subscription_id"`
	SubscriptionPubID string    `json:"subscription_public_id"`
	CustomerID        uuid.UUID `json:"customer_id"`
	CustomerPubID     string    `json:"customer_public_id"`
	InvoiceID         uuid.UUID `json:"invoice_id"`
	InvoicePubID      string    `json:"invoice_public_id"`
	InvoiceAmount     int64     `json:"invoice_amount"`
	InvoiceCurrency   string    `json:"invoice_currency"`
	InvoiceStatus     string    `json:"invoice_status"`
	NewPeriodEnd      time.Time `json:"new_period_end"`
	RenewedAt         time.Time `json:"renewed_at"`
}
