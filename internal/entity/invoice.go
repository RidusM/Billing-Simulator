package entity

import (
	"time"

	"github.com/google/uuid"
)

type InvoiceStatus string

const (
	InvoiceStatusDraft         InvoiceStatus = "draft"
	InvoiceStatusOpen          InvoiceStatus = "open"
	InvoiceStatusPaid          InvoiceStatus = "paid"
	InvoiceStatusUncollectible InvoiceStatus = "uncollectible"
	InvoiceStatusVoid          InvoiceStatus = "void"
)

type Invoice struct {
	ID                   uuid.UUID
	PublicID             string
	SubscriptionID       *uuid.UUID
	SubscriptionPublicID *string
	CustomerID           uuid.UUID
	CustomerPublicID     string
	Amount               int64
	AmountPaid           int64
	AmountRemaining      int64
	Currency             string
	Status               InvoiceStatus
	PeriodStart          *time.Time
	PeriodEnd            *time.Time
	DueDate              *time.Time
	AttemptCount         int
	AttemptedAt          *time.Time
	HostedInvoiceURL     string
	InvoicePDFURL        string
	Metadata             map[string]string
	DeletedAt            *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time

	AggregateRoot
}

func NewInvoice(
	customerID uuid.UUID,
	customerPublicID string,
	subscriptionID *uuid.UUID,
	subscriptionPublicID *string,
	amount int64,
	currency string,
	now time.Time,
) (*Invoice, error) {
	pubID, err := GeneratePublicID("inv")
	if err != nil {
		return nil, err
	}

	utc := now.UTC()

	inv := &Invoice{
		ID:                   uuid.New(),
		PublicID:             pubID,
		CustomerID:           customerID,
		CustomerPublicID:     customerPublicID,
		SubscriptionID:       subscriptionID,
		SubscriptionPublicID: subscriptionPublicID,
		Amount:               amount,
		AmountRemaining:      amount,
		Currency:             currency,
		Status:               InvoiceStatusOpen,
		Metadata:             NewMetadata(),
		CreatedAt:            utc,
		UpdatedAt:            utc,
	}

	inv.domainEvents.Raise(InvoiceCreatedEvent{
		InvoiceID:         inv.ID,
		InvoicePubID:      inv.PublicID,
		CustomerID:        inv.CustomerID,
		CustomerPubID:     inv.CustomerPublicID,
		SubscriptionID:    subscriptionID,
		SubscriptionPubID: subscriptionPublicID,
		Amount:            inv.Amount,
		Currency:          inv.Currency,
		Status:            inv.Status,
		CreatedAt:         utc,
	})

	return inv, nil
}

func NewRenewalInvoice(
	sub *Subscription,
	price *Price,
	now time.Time,
) (*Invoice, error) {

	utc := now.UTC()

	publicID, err := GeneratePublicID("inv")
	if err != nil {
		return nil, err
	}

	periodStart := sub.CurrentPeriodEnd
	periodEnd := price.NextBillingDate(periodStart)

	inv := &Invoice{
		ID:                   uuid.New(),
		PublicID:             publicID,
		CustomerID:           sub.CustomerID,
		CustomerPublicID:     sub.CustomerPublicID,
		SubscriptionID:       &sub.ID,
		SubscriptionPublicID: &sub.PublicID,
		Amount:               price.Amount,
		AmountRemaining:      price.Amount,
		Currency:             price.Currency,
		Status:               InvoiceStatusOpen,

		PeriodStart: &periodStart,
		PeriodEnd:   &periodEnd,
		DueDate:     &periodStart,

		Metadata:  make(map[string]string),
		CreatedAt: utc,
		UpdatedAt: utc,
	}

	inv.domainEvents.Raise(InvoiceCreatedEvent{
		InvoiceID:         inv.ID,
		InvoicePubID:      inv.PublicID,
		CustomerID:        inv.CustomerID,
		CustomerPubID:     inv.CustomerPublicID,
		SubscriptionID:    inv.SubscriptionID,
		SubscriptionPubID: inv.SubscriptionPublicID,
		Amount:            inv.Amount,
		Currency:          inv.Currency,
		Status:            inv.Status,
		CreatedAt:         utc,
	})

	return inv, nil
}

func (i *Invoice) MarkPaid(now time.Time) error {
	if !i.CanBePaid() {
		if i.Status == InvoiceStatusPaid {
			return ErrInvoiceAlreadyPaid
		}
		return ErrInvoiceNotPayable
	}

	utc := now.UTC()

	i.Status = InvoiceStatusPaid
	i.AmountPaid = i.Amount
	i.AttemptedAt = &utc
	i.AmountRemaining = 0
	i.UpdatedAt = utc

	i.domainEvents.Raise(InvoicePaidEvent{
		InvoiceID:      i.ID,
		InvoicePubID:   i.PublicID,
		CustomerID:     i.CustomerID,
		CustomerPubID:  i.CustomerPublicID,
		SubscriptionID: i.SubscriptionID,
		Amount:         i.Amount,
		Currency:       i.Currency,
		PaidAt:         utc,
	})

	return nil
}

func (i *Invoice) MarkPaymentFailed(now time.Time, errorCode string, isFinalAttempt bool) error {
	i.AttemptCount++
	utc := now.UTC()
	i.AttemptedAt = &utc
	i.UpdatedAt = utc

	if isFinalAttempt {
		i.Status = InvoiceStatusUncollectible
	} else {
		i.Status = InvoiceStatusOpen
	}

	i.domainEvents.Raise(InvoicePaymentFailedEvent{
		InvoiceID:      i.ID,
		InvoicePubID:   i.PublicID,
		CustomerID:     i.CustomerID,
		CustomerPubID:  i.CustomerPublicID,
		SubscriptionID: i.SubscriptionID,
		Amount:         i.Amount,
		Currency:       i.Currency,
		ErrorCode:      errorCode,
		FailedAt:       utc,
	})

	return nil
}

func (i *Invoice) CanBePaid() bool {
	return i.Status == InvoiceStatusOpen || i.Status == InvoiceStatusDraft
}

func (i *Invoice) GetAndClearEvents() DomainEvents {
	return i.domainEvents.ClearAndReturn()
}
