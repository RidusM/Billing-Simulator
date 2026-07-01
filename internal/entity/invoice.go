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
