package http

import (
	"time"

	"github.com/google/uuid"
)

// swagger:model CreateCustomerRequest
type CreateCustomerRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// swagger:model CreateSubscriptionRequest
type CreateSubscriptionRequest struct {
	CustomerID uuid.UUID `json:"customer_id" binding:"required"`
	PriceID    string    `json:"price_id" binding:"required"`
}

// swagger:model CustomerResponse
type CustomerResponse struct {
	ID        uuid.UUID `json:"id"`
	PublicID  string    `json:"public_id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// swagger:model SubscriptionResponse
type SubscriptionResponse struct {
	ID               uuid.UUID `json:"id"`
	PublicID         string    `json:"public_id"`
	CustomerID       uuid.UUID `json:"customer_id"`
	Status           string    `json:"status"`
	PriceID          string    `json:"price_id"`
	CurrentPeriodEnd time.Time `json:"current_period_end"`
	NextBillingAt    time.Time `json:"next_billing_at"`
}

// swagger:model AdvanceTimeRequest
type AdvanceTimeRequest struct {
	Duration time.Duration `json:"duration" binding:"required"` // например, 720 * time.Hour для 30 дней
}

// swagger:model BaseResponse
type BaseResponse struct {
	Data any `json:"data"`
}

// swagger:model ErrorResponse
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// swagger:model HealthResponse
type HealthResponse struct {
	Status  string    `json:"status"`
	Service string    `json:"service"`
	Time    time.Time `json:"time"`
}
