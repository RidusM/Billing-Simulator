package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type SubscriptionStatus string

const (
	SubscriptionStatusActive     SubscriptionStatus = "active"
	SubscriptionStatusPastDue    SubscriptionStatus = "past_due"
	SubscriptionStatusCanceled   SubscriptionStatus = "canceled"
	SubscriptionStatusUnpaid     SubscriptionStatus = "unpaid"
	SubscriptionStatusTrialing   SubscriptionStatus = "trialing"
	SubscriptionStatusIncomplete SubscriptionStatus = "incomplete"
)

type Subscription struct {
	ID                  uuid.UUID
	PublicID            string
	CustomerID          uuid.UUID
	CustomerPublicID    string // ← Добавлено для событий
	PriceID             uuid.UUID
	PricePublicID       string // ← Добавлено для событий
	Status              SubscriptionStatus
	CurrentPeriodStart  time.Time
	CurrentPeriodEnd    time.Time
	NextBillingAt       time.Time
	TrialStart          *time.Time
	TrialEnd            *time.Time
	CanceledAt          *time.Time
	CancelAtPeriodEnd   bool
	CancellationDetails json.RawMessage
	Metadata            map[string]string
	DeletedAt           *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time

	domainEvents DomainEvents
}

func NewSubscription(customerID, priceID uuid.UUID, periodStart, periodEnd time.Time, now time.Time) *Subscription {
	pubID, _ := GeneratePublicID("sub")
	return &Subscription{
		ID:                 uuid.New(),
		PublicID:           pubID,
		CustomerID:         customerID,
		PriceID:            priceID,
		Status:             SubscriptionStatusActive,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		NextBillingAt:      periodEnd,
		Metadata:           make(map[string]string),
		CreatedAt:          now,
		UpdatedAt:          now,
		domainEvents:       make(DomainEvents, 0),
	}
}

func (s *Subscription) Renew(now time.Time, price *Price) (*Invoice, error) {
	if s.Status != SubscriptionStatusActive && s.Status != SubscriptionStatusTrialing {
		return nil, ErrSubscriptionNotActive
	}

	invoice := NewInvoice(
		s.CustomerID,
		s.CustomerPublicID,
		&s.ID,
		&s.PublicID,
		price.Amount,
		price.Currency,
		now,
	)

	s.CurrentPeriodStart = s.CurrentPeriodEnd
	s.CurrentPeriodEnd = price.NextBillingDate(s.CurrentPeriodEnd)
	s.NextBillingAt = s.CurrentPeriodEnd
	s.Status = SubscriptionStatusActive
	s.UpdatedAt = now

	// Заполняем событие ПОЛНОЙ информацией
	s.domainEvents = append(s.domainEvents, SubscriptionRenewedEvent{
		SubscriptionID:    s.ID,
		SubscriptionPubID: s.PublicID,
		CustomerID:        s.CustomerID,
		CustomerPubID:     s.CustomerPublicID,
		InvoiceID:         invoice.ID,
		InvoicePubID:      invoice.PublicID,
		InvoiceAmount:     invoice.Amount,
		InvoiceCurrency:   invoice.Currency,
		InvoiceStatus:     invoice.Status,
		NewPeriodEnd:      s.CurrentPeriodEnd,
		RenewedAt:         now,
	})

	return invoice, nil
}

func (s *Subscription) Cancel(now time.Time, atPeriodEnd bool) error {
	if s.Status == SubscriptionStatusCanceled {
		return ErrSubscriptionAlreadyCanceled
	}

	if atPeriodEnd {
		s.CancelAtPeriodEnd = true
	} else {
		s.Status = SubscriptionStatusCanceled
		s.CanceledAt = &now
	}
	s.UpdatedAt = now

	// Заполняем событие ПОЛНОЙ информацией
	s.domainEvents = append(s.domainEvents, SubscriptionCanceledEvent{
		SubscriptionID:    s.ID,
		SubscriptionPubID: s.PublicID,
		CustomerID:        s.CustomerID,
		CustomerPubID:     s.CustomerPublicID,
		Status:            s.Status,
		CanceledAt:        now,
		AtPeriodEnd:       atPeriodEnd,
	})

	return nil
}

func (s *Subscription) GetAndClearEvents() DomainEvents {
	events := s.domainEvents
	s.domainEvents = nil
	return events
}

func (s *Subscription) MarkPaid(now time.Time) {
	s.Status = SubscriptionStatusActive
	s.UpdatedAt = now
}

func (s *Subscription) MarkPastDue(now time.Time) {
	s.Status = SubscriptionStatusPastDue
	s.UpdatedAt = now
}

func (s *Subscription) MarkCanceled(now time.Time) {
	s.Status = SubscriptionStatusCanceled
	s.CanceledAt = &now
	s.UpdatedAt = now
}
