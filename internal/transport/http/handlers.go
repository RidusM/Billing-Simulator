package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *BillingHandler) CreateCustomer(c *gin.Context) {
	var req CreateCustomerReqeust
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

func (h *BillingHandler) CancelSubscription(c *gin.Context) {
	idStr := c.Param("id")

	if err := h.svc.CancelSubscription(c.Request.Context(), idStr); err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.respondJSON(c, http.StatusOK, gin.H{"status": "canceled"})
}

func (h *BillingHandler) Health(c *gin.Context) {
	h.respondJSON(c, http.StatusOK, HealthResponse{
		Status:  "ok",
		Service: "auth-service",
		Time:    time.Now(),
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
