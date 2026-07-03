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
	ID             uuid.UUID
	PublicID       string
	SubscriptionID *uuid.UUID
	CustomerID     uuid.UUID
	Amount         int64
	Currency       string
	Status         InvoiceStatus
	AttemptCount   int
	CreatedAt      time.Time
}

func NewInvoice(customerID uuid.UUID, subscriptionID *uuid.UUID, amount int64, currency string) *Invoice {
    return &Invoice{
        ID:             uuid.New(),
        PublicID:       generatePublicID("inv"),
        CustomerID:     customerID,
        SubscriptionID: subscriptionID,
        Amount:         amount,
        Currency:       currency,
        Status:         InvoiceStatusOpen,
        AttemptCount:   0,
        CreatedAt:      time.Now().UTC(),
    }
}

func (i *Invoice) CanBePaid() bool {
    return i.Status == InvoiceStatusOpen || i.Status == InvoiceStatusDraft
}

func (i *Invoice) MarkPaid() error {
    if !i.CanBePaid() {
        if i.Status == InvoiceStatusPaid {
            return ErrInvoiceAlreadyPaid
        }
        return ErrInvoiceNotPayable
    }
    i.Status = InvoiceStatusPaid
    return nil
}

func (i *Invoice) MarkUncollectible() error {
    if i.Status == InvoiceStatusPaid {
        return ErrInvoiceAlreadyPaid
    }
    i.Status = InvoiceStatusUncollectible
    return nil
}

func (i *Invoice) IncrementAttempt() {
    i.AttemptCount++
}