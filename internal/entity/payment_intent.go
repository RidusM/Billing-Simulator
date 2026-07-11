package entity

import (
	"encoding/json"
	"fmt"
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

	AggregateRoot
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
	}, nil
}

func (pi *PaymentIntent) MarkSucceeded(now time.Time) {
	utc := now.UTC()

	pi.Status = PaymentIntentStatusSucceeded
	pi.AmountCaptured = pi.Amount
	pi.UpdatedAt = utc

	pi.Raise(PaymentIntentSucceededEvent{
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

	pi.Raise(PaymentIntentFailedEvent{
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

// ✅ ДОБАВИТЬ В payment_intent.go:
func (pi *PaymentIntent) Capture(amount int64, now time.Time) error {
	if pi.Status != PaymentIntentStatusRequiresCapture &&
		pi.Status != PaymentIntentStatusProcessing {
		return fmt.Errorf("cannot capture payment in status %s", pi.Status)
	}

	if amount <= 0 || amount > (pi.Amount-pi.AmountCaptured) {
		return ErrInvalidCaptureAmount
	}

	utc := now.UTC()
	pi.AmountCaptured += amount
	pi.UpdatedAt = utc

	if pi.AmountCaptured == pi.Amount {
		pi.Status = PaymentIntentStatusSucceeded
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

	return nil
}

func (pi *PaymentIntent) CanCapture() bool {
	return pi.AmountCaptured < pi.Amount &&
		(pi.Status == PaymentIntentStatusRequiresCapture ||
			pi.Status == PaymentIntentStatusProcessing)
}
