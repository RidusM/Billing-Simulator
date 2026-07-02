package service

import (
	"time"

	"bill-stripe-sim/internal/entity"
)

type InvoiceEvent struct {
	ID             string               `json:"id"`
	PublicID       string               `json:"public_id"`
	CustomerID     string               `json:"customer_id"`
	SubscriptionID *string              `json:"subscription_id,omitempty"`
	Amount         int64                `json:"amount"`
	Currency       string               `json:"currency"`
	Status         entity.InvoiceStatus `json:"status"`
	CreatedAt      time.Time            `json:"created_at"`
}

type SubscriptionEvent struct {
	ID               string                    `json:"id"`
	PublicID         string                    `json:"public_id"`
	CustomerID       string                    `json:"customer_id"`
	Status           entity.SubscriptionStatus `json:"status"`
	PriceID          string                    `json:"price_id"`
	CurrentPeriodEnd time.Time                 `json:"current_period_end"`
	NextBillingAt    time.Time                 `json:"next_billing_at"`
	CreatedAt        time.Time                 `json:"created_at"`
}

func mapInvoiceToEvent(i *entity.Invoice) *InvoiceEvent {
	var subID *string
	if i.SubscriptionID != nil {
		s := i.SubscriptionID.String()
		subID = &s
	}

	return &InvoiceEvent{
		ID:             i.ID.String(),
		PublicID:       i.PublicID,
		CustomerID:     i.CustomerID.String(),
		SubscriptionID: subID,
		Amount:         i.Amount,
		Currency:       i.Currency,
		Status:         i.Status,
		CreatedAt:      i.CreatedAt,
	}
}

func mapSubscriptionToEvent(s *entity.Subscription) *SubscriptionEvent {
	return &SubscriptionEvent{
		ID:               s.ID.String(),
		PublicID:         s.PublicID,
		CustomerID:       s.CustomerID.String(),
		Status:           s.Status,
		PriceID:          s.PriceID,
		CurrentPeriodEnd: s.CurrentPeriodEnd,
		NextBillingAt:    s.NextBillingAt,
		CreatedAt:        s.CreatedAt,
	}
}
