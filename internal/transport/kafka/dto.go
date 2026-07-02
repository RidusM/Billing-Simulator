package kafka

import (
	"time"

	"github.com/google/uuid"
)

type InvoiceEvent struct {
	ID             uuid.UUID  `json:"id"`
	PublicID       string     `json:"public_id"`
	CustomerID     uuid.UUID  `json:"customer_id"`
	SubscriptionID *uuid.UUID `json:"subscription_id,omitempty"`
	Amount         int64      `json:"amount"`
	Currency       string     `json:"currency"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
}

type SubscriptionEvent struct {
	ID                 uuid.UUID `json:"id"`
	PublicID           string    `json:"public_id"`
	CustomerID         uuid.UUID `json:"customer_id"`
	Status             string    `json:"status"`
	PriceID            string    `json:"price_id"`
	CurrentPeriodStart time.Time `json:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`
	NextBillingAt      time.Time `json:"next_billing_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
