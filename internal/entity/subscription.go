package entity

import (
	"encoding/json"
	"fmt"
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
	CustomerPublicID    string
	PriceID             uuid.UUID
	PricePublicID       string
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
	domainEvents        DomainEvents
}

func NewSubscription(customerID, priceID uuid.UUID, customerPubID, pricePubID string, periodStart, periodEnd time.Time, now time.Time) (*Subscription, error) {
	pubID, err := GeneratePublicID("sub")
	if err != nil {
		return nil, fmt.Errorf("failed to generate subscription public id: %w", err)
	}

	utc := now.UTC()

	s := &Subscription{
		ID:                 uuid.New(),
		PublicID:           pubID,
		CustomerID:         customerID,
		CustomerPublicID:   customerPubID,
		PriceID:            priceID,
		PricePublicID:      pricePubID,
		Status:             SubscriptionStatusActive,
		CurrentPeriodStart: periodStart.UTC(),
		CurrentPeriodEnd:   periodEnd.UTC(),
		NextBillingAt:      periodEnd.UTC(),
		Metadata:           NewMetadata(),
		CreatedAt:          utc,
		UpdatedAt:          utc,
		domainEvents:       make(DomainEvents, 0),
	}

	s.domainEvents.Raise(SubscriptionCreatedEvent{
		SubscriptionID:    s.ID,
		SubscriptionPubID: s.PublicID,
		CustomerID:        s.CustomerID,
		CustomerPubID:     s.CustomerPublicID,
		PriceID:           s.PriceID,
		PricePubID:        s.PricePublicID,
		Status:            s.Status,
		CurrentPeriodEnd:  s.CurrentPeriodEnd,
		NextBillingAt:     s.NextBillingAt,
		CreatedAt:         utc,
	})

	return s, nil
}

func (s *Subscription) Cancel(now time.Time, atPeriodEnd bool) error {
	if s.Status == SubscriptionStatusCanceled {
		return ErrSubscriptionAlreadyCanceled
	}

	if atPeriodEnd {
		if s.CancelAtPeriodEnd {
			return nil
		}

		s.CancelAtPeriodEnd = true
		s.markUpdated(now)

		return nil
	}

	s.MarkCanceled(now)

	return nil
}

func (s *Subscription) markUpdated(now time.Time) {
	utc := now.UTC()

	s.UpdatedAt = utc

	s.domainEvents.Raise(SubscriptionUpdatedEvent{
		SubscriptionID:    s.ID,
		SubscriptionPubID: s.PublicID,
		CustomerID:        s.CustomerID,
		CustomerPubID:     s.CustomerPublicID,
		Status:            s.Status,
		CancelAtPeriodEnd: s.CancelAtPeriodEnd,
		UpdatedAt:         utc,
	})
}

func (s *Subscription) Renew(now time.Time, price *Price) (*Invoice, error) {

	if s.Status != SubscriptionStatusActive &&
		s.Status != SubscriptionStatusTrialing {
		return nil, ErrSubscriptionNotActive
	}

	return NewRenewalInvoice(s, price, now)
}

// ConfirmRenewal вызывается сервисом ТОЛЬКО когда платеж за инвойс успешен
func (s *Subscription) ConfirmRenewal(newPeriodEnd time.Time, invoice *Invoice, now time.Time) {
	utc := now.UTC()
	
	s.CurrentPeriodStart = s.CurrentPeriodEnd
	s.CurrentPeriodEnd = newPeriodEnd
	s.NextBillingAt = newPeriodEnd
	s.Status = SubscriptionStatusActive
	s.UpdatedAt = utc

	s.domainEvents.Raise(SubscriptionRenewedEvent{
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
		RenewedAt:         utc,
	})
}

func (s *Subscription) GetAndClearEvents() DomainEvents {
	return s.domainEvents.ClearAndReturn()
}

func (s *Subscription) MarkPaid(now time.Time) {
	if s.Status == SubscriptionStatusActive {
		return
	}

	s.Status = SubscriptionStatusActive
	s.markUpdated(now)
}

func (s *Subscription) MarkPastDue(now time.Time) {
	if s.Status == SubscriptionStatusPastDue {
		return
	}

	s.Status = SubscriptionStatusPastDue
	s.markUpdated(now)
}

func (s *Subscription) MarkCanceled(now time.Time) {
	if s.Status == SubscriptionStatusCanceled {
		return
	}

	utc := now.UTC()

	s.Status = SubscriptionStatusCanceled
	s.CanceledAt = &utc
	s.UpdatedAt = utc

	s.domainEvents.Raise(SubscriptionCanceledEvent{
		SubscriptionID:    s.ID,
		SubscriptionPubID: s.PublicID,
		CustomerID:        s.CustomerID,
		CustomerPubID:     s.CustomerPublicID,
		Status:            s.Status,
		CanceledAt:        utc,
		AtPeriodEnd:       s.CancelAtPeriodEnd,
	})
}
