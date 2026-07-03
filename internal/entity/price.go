package entity

import "time"

type BillingInterval string

const (
	BillingIntervalDay   BillingInterval = "day"
	BillingIntervalWeek  BillingInterval = "week"
	BillingIntervalMonth BillingInterval = "month"
	BillingIntervalYear  BillingInterval = "year"
)

type Price struct {
	ID            string
	Amount        int64
	Currency      string
	Interval      BillingInterval
	IntervalCount int
	CreatedAt     time.Time
}

func (p *Price) NextBillingDate(from time.Time) time.Time {
	count := p.IntervalCount
	if count <= 0 {
		count = 1
	}
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