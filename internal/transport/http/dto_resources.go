package handler

import (
	"errors"
	"net/http"
	"time"

	"bill-stripe-sim/internal/entity"

	"github.com/gin-gonic/gin"
)

// ---------- Customer / Subscription: маппинг entity -> DTO (без утечки uuid.UUID наружу) ----------

func toCustomerResponse(c *entity.Customer) CustomerResponse {
	return CustomerResponse{
		ID:        c.PublicID,
		Email:     c.Email,
		CreatedAt: c.CreatedAt,
	}
}

func toSubscriptionResponse(s *entity.Subscription) SubscriptionResponse {
	return SubscriptionResponse{
		ID:               s.PublicID,
		CustomerID:       s.CustomerPublicID,
		Status:           string(s.Status),
		PriceID:          s.PricePublicID,
		CurrentPeriodEnd: s.CurrentPeriodEnd,
		NextBillingAt:    s.NextBillingAt,
	}
}

// ---------- Product ----------

type CreateProductRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type ProductResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
}

func toProductResponse(p *entity.Product) ProductResponse {
	return ProductResponse{
		ID:          p.PublicID,
		Name:        p.Name,
		Description: p.Description,
		Active:      p.Active,
		CreatedAt:   p.CreatedAt,
	}
}

// ---------- Price ----------

type CreatePriceRequest struct {
	ProductID     string `json:"product_id" binding:"required"`
	Amount        int64  `json:"amount" binding:"required,gt=0"`
	Currency      string `json:"currency" binding:"required,len=3"`
	Interval      string `json:"interval" binding:"required,oneof=day week month year"`
	IntervalCount int    `json:"interval_count"`
}

type PriceResponse struct {
	ID            string `json:"id"`
	ProductID     string `json:"product_id"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	Interval      string `json:"interval"`
	IntervalCount int    `json:"interval_count"`
	Active        bool   `json:"active"`
}

func toPriceResponse(p *entity.Price, productPublicID string) PriceResponse {
	return PriceResponse{
		ID:            p.PublicID,
		ProductID:     productPublicID,
		Amount:        p.Amount,
		Currency:      p.Currency,
		Interval:      string(p.Interval),
		IntervalCount: p.IntervalCount,
		Active:        p.Active,
	}
}

// ---------- Invoice ----------

type InvoiceResponse struct {
	ID             string     `json:"id"`
	CustomerID     string     `json:"customer_id"`
	SubscriptionID *string    `json:"subscription_id,omitempty"`
	Amount         int64      `json:"amount"`
	AmountPaid     int64      `json:"amount_paid"`
	Currency       string     `json:"currency"`
	Status         string     `json:"status"`
	AttemptCount   int        `json:"attempt_count"`
	DueDate        *time.Time `json:"due_date,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

func toInvoiceResponse(i *entity.Invoice) InvoiceResponse {
	return InvoiceResponse{
		ID:             i.PublicID,
		CustomerID:     i.CustomerPublicID,
		SubscriptionID: i.SubscriptionPublicID,
		Amount:         i.Amount,
		AmountPaid:     i.AmountPaid,
		Currency:       i.Currency,
		Status:         string(i.Status),
		AttemptCount:   i.AttemptCount,
		DueDate:        i.DueDate,
		CreatedAt:      i.CreatedAt,
	}
}

// ---------- PaymentIntent ----------

type CreatePaymentIntentRequest struct {
	InvoiceID string `json:"invoice_id" binding:"required"`
}

type PaymentIntentResponse struct {
	ID             string `json:"id"`
	InvoiceID      string `json:"invoice_id,omitempty"`
	Amount         int64  `json:"amount"`
	AmountCaptured int64  `json:"amount_captured"`
	Currency       string `json:"currency"`
	Status         string `json:"status"`
}

func toPaymentIntentResponse(pi *entity.PaymentIntent, invoicePublicID string) PaymentIntentResponse {
	return PaymentIntentResponse{
		ID:             pi.PublicID,
		InvoiceID:      invoicePublicID,
		Amount:         pi.Amount,
		AmountCaptured: pi.AmountCaptured,
		Currency:       pi.Currency,
		Status:         string(pi.Status),
	}
}

// ---------- WebhookEndpoint ----------

type CreateWebhookEndpointRequest struct {
	CustomerID    string   `json:"customer_id" binding:"required"`
	URL           string   `json:"url" binding:"required,url"`
	Description   string   `json:"description"`
	EnabledEvents []string `json:"enabled_events"`
}

// WebhookEndpointResponse: secret возвращается ТОЛЬКО в ответе на создание (см. хендлер
// CreateWebhookEndpoint) через параметр includeSecret=true. В GET/List — только secret_prefix.
type WebhookEndpointResponse struct {
	ID            string   `json:"id"`
	URL           string   `json:"url"`
	Description   string   `json:"description"`
	SecretPrefix  string   `json:"secret_prefix"`
	Secret        string   `json:"secret,omitempty"`
	EnabledEvents []string `json:"enabled_events"`
	Enabled       bool     `json:"enabled"`
}

func toWebhookEndpointResponse(e *entity.WebhookEndpoint, includeSecret bool) WebhookEndpointResponse {
	resp := WebhookEndpointResponse{
		ID:            e.PublicID,
		URL:           e.URL,
		Description:   e.Description,
		SecretPrefix:  e.SecretPrefix,
		EnabledEvents: e.EnabledEvents,
		Enabled:       e.Enabled,
	}
	if includeSecret {
		resp.Secret = e.Secret
	}
	return resp
}

// ---------- расширение error_handling.go: доп. sentinel-ошибки, которых там не было ----------
// Вызывать из handleServiceError ДО default-ветки:
//
//	if status, code, ok := extraErrorStatus(err); ok {
//	    h.respondError(c, status, code, err.Error(), nil)
//	    return
//	}
func extraErrorStatus(err error) (status int, code string, ok bool) {
	switch {
	case errors.Is(err, entity.ErrProductNotFound),
		errors.Is(err, entity.ErrPriceNotFound),
		errors.Is(err, entity.ErrPaymentIntentNotFound),
		errors.Is(err, entity.ErrWebhookEndpointNotFound):
		return http.StatusNotFound, "resource_not_found", true
	case errors.Is(err, entity.ErrWebhookEndpointDisabled),
		errors.Is(err, entity.ErrInvalidPrice):
		return http.StatusBadRequest, "invalid_request", true
	default:
		return 0, "", false
	}
}

func respondCreated(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, BaseResponse{Data: data})
}

func respondOK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, BaseResponse{Data: data})
}
