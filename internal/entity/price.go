package entity

import (
	"time"

	"github.com/google/uuid"
)

type BillingInterval string

const (
	BillingIntervalDay   BillingInterval = "day"
	BillingIntervalWeek  BillingInterval = "week"
	BillingIntervalMonth BillingInterval = "month"
	BillingIntervalYear  BillingInterval = "year"
)

type Price struct {
	ID            uuid.UUID
	PublicID      string
	ProductID     uuid.UUID
	Amount        int64
	Currency      string
	Interval      BillingInterval
	IntervalCount int
	Active        bool
	Metadata      map[string]string
	DeletedAt     *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewPrice(productID uuid.UUID, amount int64, currency string, interval BillingInterval, intervalCount int, now time.Time) *Price {
	if intervalCount <= 0 {
		intervalCount = 1
	}
	return &Price{
		ID:            uuid.New(),
		PublicID:      GeneratePublicID("price"),
		ProductID:     productID,
		Amount:        amount,
		Currency:      currency,
		Interval:      interval,
		IntervalCount: intervalCount,
		Active:        true,
		Metadata:      make(map[string]string),
		CreatedAt:     now.UTC(),
		UpdatedAt:     now.UTC(),
	}
}

func (p *Price) NextBillingDate(from time.Time) time.Time {
	count := p.IntervalCount
	switch p.Interval {
	case BillingIntervalDay:
		return from.AddDate(0, 0, count)
	case BillingIntervalWeek:
		return from.AddDate(0, 0, 7*count)
	case BillingIntervalMonth:
		return from.AddDate(0, count, 0)
	case BillingIntervalYear:
		return from.AddDate(count, 0, 0)
	default:
		return from.AddDate(0, count, 0)
	}
}
