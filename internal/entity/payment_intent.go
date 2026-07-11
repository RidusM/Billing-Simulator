package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type PaymentIntentStatus string

const (
	PaymentIntentStatusRequiresPaymentMethod PaymentIntentStatus = "requires_payment_method"
	PaymentIntentStatusRequiresConfirmation  PaymentIntentStatus = "requires_confirmation"
	PaymentIntentStatusRequiresAction        PaymentIntentStatus = "requires_action"
	PaymentIntentStatusRequiresCapture       PaymentIntentStatus = "requires_capture"
	PaymentIntentStatusProcessing            PaymentIntentStatus = "processing"
	PaymentIntentStatusSucceeded             PaymentIntentStatus = "succeeded"
	PaymentIntentStatusCanceled              PaymentIntentStatus = "canceled"
)

type PaymentIntent struct {
	ID                uuid.UUID
	PublicID          string
	InvoiceID         *uuid.UUID
	CustomerID        uuid.UUID
	Amount            int64
	AmountCaptured    int64
	Currency          string
	Status            PaymentIntentStatus
	LastPaymentError  json.RawMessage
	PaymentMethodID   string
	PaymentMethodType string
	Metadata          map[string]string
	DeletedAt         *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
	domainEvents      DomainEvents
}

func NewPaymentIntent(customerID uuid.UUID, invoiceID *uuid.UUID, amount int64, currency string, now time.Time) *PaymentIntent {
	pubID, _ := GeneratePublicID("pi")
	return &PaymentIntent{
		ID:                uuid.New(),
		PublicID:          pubID,
		InvoiceID:         invoiceID,
		CustomerID:        customerID,
		Amount:            amount,
		Currency:          currency,
		Status:            PaymentIntentStatusRequiresPaymentMethod,
		PaymentMethodType: "card",
		Metadata:          make(map[string]string),
		CreatedAt:         now.UTC(),
		UpdatedAt:         now.UTC(),
		domainEvents:      make(DomainEvents, 0),
	}
}

func (pi *PaymentIntent) MarkSucceeded(now time.Time) {
	pi.Status = PaymentIntentStatusSucceeded
	pi.AmountCaptured = pi.Amount
	pi.UpdatedAt = now.UTC()

	pi.domainEvents.Raise(PaymentIntentSucceededEvent{
		PaymentIntentID:    pi.ID,
		PaymentIntentPubID: pi.PublicID,
		CustomerID:         pi.CustomerID,
		InvoiceID:          pi.InvoiceID,
		Amount:             pi.Amount,
		Currency:           pi.Currency,
		SucceededAt:        now.UTC(),
	})
}

func (pi *PaymentIntent) MarkFailed(now time.Time, errorCode, declineCode string) {
	pi.Status = PaymentIntentStatusRequiresPaymentMethod
	pi.UpdatedAt = now.UTC()

	bytes, _ := json.Marshal(map[string]string{
		"code":         errorCode,
		"decline_code": declineCode,
	})
	pi.LastPaymentError = bytes

	pi.domainEvents.Raise(PaymentIntentFailedEvent{
		PaymentIntentID:    pi.ID,
		PaymentIntentPubID: pi.PublicID,
		CustomerID:         pi.CustomerID,
		InvoiceID:          pi.InvoiceID,
		Amount:             pi.Amount,
		Currency:           pi.Currency,
		ErrorCode:          errorCode,
		DeclineCode:        declineCode,
		FailedAt:           now.UTC(),
	})
}

func (pi *PaymentIntent) GetAndClearEvents() DomainEvents {
	return pi.domainEvents.ClearAndReturn()
}
