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
	SubscriptionPublicID *string // ← Добавлено для событий
	CustomerID           uuid.UUID
	CustomerPublicID     string // ← Добавлено для событий
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

	domainEvents DomainEvents
}

func NewInvoice(customerID uuid.UUID, subscriptionID *uuid.UUID, amount int64, currency string, now time.Time) *Invoice {
	inv := &Invoice{
		ID:              uuid.New(),
		PublicID:        GeneratePublicID("in"),
		CustomerID:      customerID,
		SubscriptionID:  subscriptionID,
		Amount:          amount,
		AmountRemaining: amount,
		Currency:        currency,
		Status:          InvoiceStatusOpen,
		Metadata:        make(map[string]string),
		CreatedAt:       now,
		UpdatedAt:       now,
		domainEvents:    make(DomainEvents, 0),
	}

	// Заполняем событие ПОЛНОЙ информацией
	inv.domainEvents = append(inv.domainEvents, InvoiceCreatedEvent{
		InvoiceID:      inv.ID,
		InvoicePubID:   inv.PublicID,
		CustomerID:     inv.CustomerID,
		SubscriptionID: subscriptionID,
		Amount:         inv.Amount,
		Currency:       inv.Currency,
		Status:         inv.Status,
		CreatedAt:      now,
	})

	return inv
}

// SetCustomerPublicID — устанавливает публичный ID клиента для событий
func (i *Invoice) SetCustomerPublicID(publicID string) {
	i.CustomerPublicID = publicID
	// Обновляем событие создания
	for idx, event := range i.domainEvents {
		if e, ok := event.(InvoiceCreatedEvent); ok {
			e.CustomerPubID = publicID
			i.domainEvents[idx] = e
		}
	}
}

// SetSubscriptionPublicID — устанавливает публичный ID подписки для событий
func (i *Invoice) SetSubscriptionPublicID(publicID string) {
	i.SubscriptionPublicID = &publicID
	for idx, event := range i.domainEvents {
		if e, ok := event.(InvoiceCreatedEvent); ok {
			e.SubscriptionPubID = &publicID
			i.domainEvents[idx] = e
		}
	}
}

func (i *Invoice) MarkPaid(now time.Time) error {
	if !i.CanBePaid() {
		if i.Status == InvoiceStatusPaid {
			return ErrInvoiceAlreadyPaid
		}
		return ErrInvoiceNotPayable
	}

	i.Status = InvoiceStatusPaid
	i.AmountPaid = i.Amount
	i.AmountRemaining = 0
	i.UpdatedAt = now

	// Заполняем событие ПОЛНОЙ информацией
	i.domainEvents = append(i.domainEvents, InvoicePaidEvent{
		InvoiceID:      i.ID,
		InvoicePubID:   i.PublicID,
		CustomerID:     i.CustomerID,
		CustomerPubID:  i.CustomerPublicID,
		SubscriptionID: i.SubscriptionID,
		Amount:         i.Amount,
		Currency:       i.Currency,
		PaidAt:         now,
	})

	return nil
}

func (i *Invoice) MarkPaymentFailed(now time.Time, errorCode string) error {
	i.Status = InvoiceStatusUncollectible
	i.UpdatedAt = now

	// Заполняем событие ПОЛНОЙ информацией
	i.domainEvents = append(i.domainEvents, InvoicePaymentFailedEvent{
		InvoiceID:      i.ID,
		InvoicePubID:   i.PublicID,
		CustomerID:     i.CustomerID,
		CustomerPubID:  i.CustomerPublicID,
		SubscriptionID: i.SubscriptionID,
		Amount:         i.Amount,
		Currency:       i.Currency,
		ErrorCode:      errorCode,
		FailedAt:       now,
	})

	return nil
}

func (i *Invoice) CanBePaid() bool {
	return i.Status == InvoiceStatusOpen || i.Status == InvoiceStatusDraft
}

func (i *Invoice) GetAndClearEvents() DomainEvents {
	events := i.domainEvents
	i.domainEvents = nil
	return events
}
