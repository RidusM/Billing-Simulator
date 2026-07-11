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

func NewPaymentIntent(customerID uuid.UUID, invoiceID *uuid.UUID, amount int64, currency string, now time.Time) (*PaymentIntent, error) {
	pubID, err := GeneratePublicID("pi")
	if err != nil {
		return nil, err
	}

	utc := now.UTC()

	return &PaymentIntent{
		ID:                uuid.New(),
		PublicID:          pubID,
		InvoiceID:         invoiceID,
		CustomerID:        customerID,
		Amount:            amount,
		Currency:          currency,
		Status:            PaymentIntentStatusRequiresPaymentMethod,
		PaymentMethodType: "card",
		Metadata:          NewMetadata(),
		CreatedAt:         utc,
		UpdatedAt:         utc,
		domainEvents:      make(DomainEvents, 0),
	}, nil
}

func (pi *PaymentIntent) MarkSucceeded(now time.Time) {
	utc := now.UTC()

	pi.Status = PaymentIntentStatusSucceeded
	pi.AmountCaptured = pi.Amount
	pi.UpdatedAt = utc

	pi.domainEvents.Raise(PaymentIntentSucceededEvent{
		PaymentIntentID:    pi.ID,
		PaymentIntentPubID: pi.PublicID,
		CustomerID:         pi.CustomerID,
		InvoiceID:          pi.InvoiceID,
		Amount:             pi.Amount,
		Currency:           pi.Currency,
		SucceededAt:        utc,
	})
}

func (pi *PaymentIntent) MarkFailed(now time.Time, errorCode, declineCode string) {
	utc := now.UTC()

	pi.Status = PaymentIntentStatusRequiresPaymentMethod
	pi.UpdatedAt = utc

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
		FailedAt:           utc,
	})
}

func (pi *PaymentIntent) GetAndClearEvents() DomainEvents {
	return pi.domainEvents.ClearAndReturn()
}
