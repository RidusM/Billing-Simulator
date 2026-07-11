package handler

import (
	"time"
)

// ЗАМЕНЯЕТ ваш http/dto.go целиком.
// Главное изменение: наружу (в запросах и ответах) никогда не торчит внутренний uuid.UUID —
// только public_id-строки ("cus_xxx", "price_xxx", "sub_xxx"), как в реальном Stripe API.

// swagger:model CreateCustomerRequest
type CreateCustomerRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// swagger:model CreateSubscriptionRequest
type CreateSubscriptionRequest struct {
	CustomerID string `json:"customer_id" binding:"required"` // public_id, напр. "cus_1a2b3c4d"
	PriceID    string `json:"price_id" binding:"required"`    // public_id, напр. "price_1a2b3c4d"
}

// swagger:model CancelSubscriptionRequest
type CancelSubscriptionRequest struct {
	AtPeriodEnd bool `json:"at_period_end"`
}

// swagger:model CustomerResponse
type CustomerResponse struct {
	ID        string    `json:"id"` // public_id
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// swagger:model SubscriptionResponse
type SubscriptionResponse struct {
	ID               string    `json:"id"` // public_id
	CustomerID       string    `json:"customer_id"`
	Status           string    `json:"status"`
	PriceID          string    `json:"price_id"`
	CurrentPeriodEnd time.Time `json:"current_period_end"`
	NextBillingAt    time.Time `json:"next_billing_at"`
}

// swagger:model AdvanceTimeRequest
type AdvanceTimeRequest struct {
	Duration time.Duration `json:"duration" binding:"required"` // напр. 720 * time.Hour для 30 дней
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
