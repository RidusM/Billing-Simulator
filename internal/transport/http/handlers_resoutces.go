package handler

import (
	"errors"
	"net/http"

	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/internal/service"
	"bill-stripe-sim/pkg/logger"

	"github.com/gin-gonic/gin"
)

// ResourceHandlers — второй хендлер-блок поверх тех же концепций, что и BillingHandler,
// но для ресурсов, которые не завязаны на единый facade (Product/Price/Invoice/PaymentIntent/
// WebhookEndpoint у каждого свой конкретный сервис 1:1, facade им не нужен).
type ResourceHandlers struct {
	products  *service.ProductService
	prices    *service.PriceService
	invoices  *service.InvoiceQueryService
	payments  *service.PaymentService
	webhooks  *service.WebhookEndpointService
	customers *service.CustomerService
	log       logger.Logger
}

func NewResourceHandlers(
	products *service.ProductService,
	prices *service.PriceService,
	invoices *service.InvoiceQueryService,
	payments *service.PaymentService,
	webhooks *service.WebhookEndpointService,
	customers *service.CustomerService,
	log logger.Logger,
) *ResourceHandlers {
	return &ResourceHandlers{
		products:  products,
		prices:    prices,
		invoices:  invoices,
		payments:  payments,
		webhooks:  webhooks,
		customers: customers,
		log:       log,
	}
}

// RegisterRoutes монтирует все ресурсные роуты на уже существующий /v1 group.
// Вызывать из composition root: handler.NewResourceHandlers(...).RegisterRoutes(v1)
func (h *ResourceHandlers) RegisterRoutes(v1 *gin.RouterGroup) {
	products := v1.Group("/products")
	{
		products.POST("", h.CreateProduct)
		products.GET("/:id", h.GetProduct)
	}

	prices := v1.Group("/prices")
	{
		prices.POST("", h.CreatePrice)
		prices.GET("/:id", h.GetPrice)
	}

	invoices := v1.Group("/invoices")
	{
		invoices.GET("/:id", h.GetInvoice)
	}

	customers := v1.Group("/customers")
	{
		customers.GET("/:id/invoices", h.ListCustomerInvoices)
		customers.POST("/:id/webhook_endpoints", h.CreateWebhookEndpoint)
		customers.GET("/:id/webhook_endpoints", h.ListWebhookEndpoints)
	}

	paymentIntents := v1.Group("/payment_intents")
	{
		paymentIntents.POST("", h.CreatePaymentIntent)
		paymentIntents.POST("/:id/confirm", h.ConfirmPaymentIntent)
	}
}

// ---------- Product ----------

// @Summary  Create product
// @Tags     products
// @Router   /v1/products [post]
func (h *ResourceHandlers) CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	product, err := h.products.CreateProduct(c.Request.Context(), req.Name, req.Description)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	respondCreated(c, toProductResponse(product))
}

// @Summary  Get product
// @Tags     products
// @Router   /v1/products/{id} [get]
func (h *ResourceHandlers) GetProduct(c *gin.Context) {
	product, err := h.products.GetProduct(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	respondOK(c, toProductResponse(product))
}

// ---------- Price ----------

// @Summary  Create price
// @Tags     prices
// @Router   /v1/prices [post]
func (h *ResourceHandlers) CreatePrice(c *gin.Context) {
	var req CreatePriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	product, err := h.products.GetProduct(c.Request.Context(), req.ProductID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	intervalCount := req.IntervalCount
	if intervalCount <= 0 {
		intervalCount = 1
	}

	price, err := h.prices.CreatePrice(
		c.Request.Context(),
		product.ID,
		req.Amount,
		req.Currency,
		entity.BillingInterval(req.Interval),
		intervalCount,
	)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	respondCreated(c, toPriceResponse(price, product.PublicID))
}

// @Summary  Get price
// @Tags     prices
// @Router   /v1/prices/{id} [get]
//
// TODO: price.ProductID — внутренний uuid, а не public_id. Чтобы отдать в ответе правильный
// "prod_xxx", добавьте в ProductService недостающий read-метод по внутреннему id:
//
//	func (s *ProductService) GetByID(ctx context.Context, id uuid.UUID) (*entity.Product, error) {
//	    return s.products.GetByID(ctx, id)
//	}
//
// и замените строку ниже на product.PublicID (см. как это уже сделано в CreatePrice).
func (h *ResourceHandlers) GetPrice(c *gin.Context) {
	price, err := h.prices.GetPriceByPublicID(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	respondOK(c, toPriceResponse(price, price.ProductID.String())) // временно: internal uuid, см. TODO выше
}

// ---------- Invoice ----------

// @Summary  Get invoice
// @Tags     invoices
// @Router   /v1/invoices/{id} [get]
func (h *ResourceHandlers) GetInvoice(c *gin.Context) {
	inv, err := h.invoices.GetInvoice(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	respondOK(c, toInvoiceResponse(inv))
}

// @Summary  List invoices for a customer
// @Tags     invoices
// @Router   /v1/customers/{id}/invoices [get]
func (h *ResourceHandlers) ListCustomerInvoices(c *gin.Context) {
	customer, err := h.customers.GetCustomer(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	invoices, err := h.invoices.ListForCustomer(c.Request.Context(), customer.ID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	resp := make([]InvoiceResponse, 0, len(invoices))
	for _, inv := range invoices {
		resp = append(resp, toInvoiceResponse(inv))
	}
	respondOK(c, resp)
}

// ---------- PaymentIntent ----------

// @Summary  Create payment intent
// @Tags     payment_intents
// @Router   /v1/payment_intents [post]
func (h *ResourceHandlers) CreatePaymentIntent(c *gin.Context) {
	var req CreatePaymentIntentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	inv, err := h.invoices.GetInvoice(c.Request.Context(), req.InvoiceID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	pi, err := h.payments.CreatePaymentIntent(c.Request.Context(), inv.ID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	respondCreated(c, toPaymentIntentResponse(pi, inv.PublicID))
}

// @Summary  Confirm payment intent (симулирует исход платежа)
// @Tags     payment_intents
// @Router   /v1/payment_intents/{id}/confirm [post]
func (h *ResourceHandlers) ConfirmPaymentIntent(c *gin.Context) {
	pi, err := h.payments.ConfirmPayment(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	invoiceID := ""
	if pi.InvoiceID != nil {
		invoiceID = pi.InvoiceID.String() // TODO: заменить на invoice.PublicID, если понадобится в ответе
	}
	respondOK(c, toPaymentIntentResponse(pi, invoiceID))
}

// ---------- WebhookEndpoint ----------

// @Summary  Register webhook endpoint for a customer
// @Tags     webhook_endpoints
// @Router   /v1/customers/{id}/webhook_endpoints [post]
func (h *ResourceHandlers) CreateWebhookEndpoint(c *gin.Context) {
	customer, err := h.customers.GetCustomer(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	var req CreateWebhookEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ep, err := h.webhooks.CreateEndpoint(c.Request.Context(), customer.ID, req.URL, req.Description, req.EnabledEvents)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	// includeSecret=true — ЕДИНСТВЕННЫЙ раз, когда полный secret уходит клиенту.
	respondCreated(c, toWebhookEndpointResponse(ep, true))
}

// @Summary  List webhook endpoints for a customer
// @Tags     webhook_endpoints
// @Router   /v1/customers/{id}/webhook_endpoints [get]
func (h *ResourceHandlers) ListWebhookEndpoints(c *gin.Context) {
	customer, err := h.customers.GetCustomer(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	endpoints, err := h.webhooks.ListForCustomer(c.Request.Context(), customer.ID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	resp := make([]WebhookEndpointResponse, 0, len(endpoints))
	for _, ep := range endpoints {
		resp = append(resp, toWebhookEndpointResponse(ep, false)) // includeSecret=false — список никогда не палит секреты
	}
	respondOK(c, resp)
}

// ---------- общий error handling для этого блока хендлеров ----------

func (h *ResourceHandlers) respondError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{Code: code, Message: message})
}

func (h *ResourceHandlers) respondServiceError(c *gin.Context, err error) {
	if status, code, ok := extraErrorStatus(err); ok {
		h.respondError(c, status, code, err.Error())
		return
	}

	switch {
	case errors.Is(err, entity.ErrDataNotFound),
		errors.Is(err, entity.ErrCustomerNotFound):
		h.respondError(c, http.StatusNotFound, "resource_not_found", err.Error())
	case errors.Is(err, entity.ErrConflictingData):
		h.respondError(c, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, entity.ErrInvalidData):
		h.respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
	default:
		h.log.LogAttrs(c.Request.Context(), logger.ErrorLevel, "unhandled service error", logger.Any("error", err))
		h.respondError(c, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
	}
}
