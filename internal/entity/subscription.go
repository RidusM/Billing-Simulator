package entity

import (
	"time"

	"github.com/google/uuid"
)

type SubscriptionStatus string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusPastDue  SubscriptionStatus = "past_due"
	SubscriptionStatusCanceled SubscriptionStatus = "canceled"
	SubscriptionStatusUnpaid   SubscriptionStatus = "unpaid"
)

type Subscription struct {
	ID                 uuid.UUID
	PublicID           string
	CustomerID         uuid.UUID
	Status             SubscriptionStatus
	PriceID            string
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
	NextBillingAt      time.Time
	CanceledAt         *time.Time
	CreatedAt          time.Time
}

func NewSubscription(customerID uuid.UUID, priceID string, periodStart, periodEnd time.Time) *Subscription {
    return &Subscription{
        ID:                 uuid.New(),
        PublicID:           generatePublicID("sub"),
        CustomerID:         customerID,
        Status:             SubscriptionStatusActive,
        PriceID:            priceID,
        CurrentPeriodStart: periodStart,
        CurrentPeriodEnd:   periodEnd,
        NextBillingAt:      periodEnd,
        CreatedAt:          time.Now().UTC(),
    }
}

func (s *Subscription) IsRenewalDue(now time.Time) bool {
    return s.Status == SubscriptionStatusActive && !s.NextBillingAt.After(now)
}

func (s *Subscription) IsPastDue(now time.Time, gracePeriod time.Duration) bool {
    return s.Status == SubscriptionStatusActive &&
        s.NextBillingAt.Before(now.Add(-gracePeriod))
}

func (s *Subscription) Renew(now time.Time, price *Price) (*Invoice, error) {
    if s.Status != SubscriptionStatusActive {
        return nil, ErrSubscriptionNotActive
    }

    invoice := NewInvoice(s.CustomerID, &s.ID, price.Amount, price.Currency)

    nextEnd := price.NextBillingDate(s.CurrentPeriodEnd)
    s.CurrentPeriodStart = s.CurrentPeriodEnd
    s.CurrentPeriodEnd = nextEnd
    s.NextBillingAt = nextEnd

    return invoice, nil
}

func (s *Subscription) Cancel(now time.Time) error {
    if s.Status == SubscriptionStatusCanceled {
        return ErrSubscriptionAlreadyCanceled
    }
    s.Status = SubscriptionStatusCanceled
    s.CanceledAt = &now
    return nil
}

func (s *Subscription) MarkPastDue() error {
    if s.Status != SubscriptionStatusActive {
        return ErrSubscriptionNotActive
    }
    s.Status = SubscriptionStatusPastDue
    return nil
}