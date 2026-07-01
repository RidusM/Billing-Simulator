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
