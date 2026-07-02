package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// @Summary      Create customer
// @Description  Registers a new customer by email for subsequent invoicing
// @Tags         customers
// @Accept       json
// @Produce      json
// @Param        request  body      CreateCustomerRequest  true  "Customer data"
// @Success      201      {object}  BaseResponse{data=CustomerResponse}
// @Failure      400      {object}  ErrorResponse
// @Failure      409      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Router       /v1/customers [post]
func (h *BillingHandler) CreateCustomer(c *gin.Context) {
	var req CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}

	customer, err := h.svc.CreateCustomer(c.Request.Context(), req.Email)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.respondJSON(c, http.StatusCreated, BaseResponse{
		Data: customer,
	})
}

// @Summary      Create subscription
// @Description  Binds the customer to a specific service plan (price_id)
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        request  body      CreateSubscriptionRequest  true  "Subscription data"
// @Success      201      {object}  BaseResponse{data=SubscriptionResponse}
// @Failure      400      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Router       /v1/subscriptions [post]
func (h *BillingHandler) CreateSubscription(c *gin.Context) {
	var req CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}

	sub, err := h.svc.CreateSubscription(c.Request.Context(), req.CustomerID, req.PriceID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.respondJSON(c, http.StatusCreated, BaseResponse{
		Data: sub,
	})
}

// @Summary      Get subscription
// @Description  Returns full subscription data, including status and period dates
// @Tags         subscriptions
// @Produce      json
// @Param        id   path      string  true  "Subscription UUID"
// @Success      200  {object}  BaseResponse{data=SubscriptionResponse}
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /v1/subscriptions/{id} [get]
func (h *BillingHandler) GetSubscription(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_uuid", "invalid subscription uuid", nil)
		return
	}

	sub, err := h.svc.GetSubscription(c.Request.Context(), id)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.respondJSON(c, http.StatusOK, BaseResponse{
		Data: sub,
	})
}

// @Summary      Cancel Subscription
// @Description  Changes the subscription status to 'canceled'. Further write-offs are stopped.
// @Tags         subscriptions
// @Produce      json
// @Param        id   path      string  true  "UUID Subscription"
// @Success      200  {object}  map[string]string "{"status": "canceled"}"
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /v1/subscriptions/{id}/cancel [post]
func (h *BillingHandler) CancelSubscription(c *gin.Context) {
	idStr := c.Param("id")

	if err := h.svc.CancelSubscription(c.Request.Context(), idStr); err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.respondJSON(c, http.StatusOK, gin.H{"status": "canceled"})
}

// @Summary      Health Check
// @Description Return service status and current timestamp. No authentication required.
// @Tags         system
// @Produce      json
// @Success      200  {object}  HealthResponse "Service is healthy"
// @Router       /health [get]
func (h *BillingHandler) Health(c *gin.Context) {
	h.respondJSON(c, http.StatusOK, HealthResponse{
		Status: "ok",
		Time:   time.Now(),
	})
}

func (h *BillingHandler) respondJSON(c *gin.Context, status int, data any) {
	c.JSON(status, data)
}

func (h *BillingHandler) respondError(c *gin.Context, status int, code, message string, details any) {
	c.AbortWithStatusJSON(status, ErrorResponse{
		Code:    code,
		Message: message,
		Details: details,
	})
}
