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
	}
}

func (pi *PaymentIntent) MarkSucceeded() {
	pi.Status = PaymentIntentStatusSucceeded
	pi.AmountCaptured = pi.Amount
}

func (pi *PaymentIntent) MarkFailed(errorCode, declineCode string) {
	pi.Status = PaymentIntentStatusRequiresPaymentMethod

	bytes, _ := json.Marshal(map[string]string{
		"code":         errorCode,
		"decline_code": declineCode,
	})

	pi.LastPaymentError = bytes
}
