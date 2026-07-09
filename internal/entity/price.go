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
	domainEvents  DomainEvents
}

func NewPrice(productID uuid.UUID, amount int64, currency string, interval BillingInterval, intervalCount int, now time.Time) *Price {
	pubID, _ := GeneratePublicID("price")
	if intervalCount <= 0 {
		intervalCount = 1
	}

	p := &Price{
		ID:            uuid.New(),
		PublicID:      pubID,
		ProductID:     productID,
		Amount:        amount,
		Currency:      currency,
		Interval:      interval,
		IntervalCount: intervalCount,
		Active:        true,
		Metadata:      make(map[string]string),
		CreatedAt:     now.UTC(),
		UpdatedAt:     now.UTC(),
		domainEvents:  make(DomainEvents, 0),
	}

	p.domainEvents = append(p.domainEvents, PriceCreatedEvent{
		PriceID:       p.ID,
		PricePubID:    p.PublicID,
		ProductID:     p.ProductID,
		Amount:        p.Amount,
		Currency:      p.Currency,
		Interval:      p.Interval,
		IntervalCount: p.IntervalCount,
		CreatedAt:     now,
	})

	return p
}

func (p Price) NextBillingDate(from time.Time) time.Time {
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

func (p *Price) Update(amount int64, active bool, now time.Time) {
	p.Amount = amount
	p.Active = active
	p.UpdatedAt = now

	p.domainEvents = append(p.domainEvents, PriceUpdatedEvent{
		PriceID:       p.ID,
		PricePubID:    p.PublicID,
		ProductID:     p.ProductID,
		Amount:        p.Amount,
		Currency:      p.Currency,
		Interval:      p.Interval,
		IntervalCount: p.IntervalCount,
		Active:        p.Active,
		UpdatedAt:     now,
	})
}

func (p *Price) GetAndClearEvents() DomainEvents {
	events := p.domainEvents
	p.domainEvents = nil
	return events
}
