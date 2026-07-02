package http

import (
	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *BillingHandler) handleServiceError(c *gin.Context, err error) {
	var statusCode int
	var errorCode string
	var message string

	switch {
	case errors.Is(err, entity.ErrDataNotFound),
		errors.Is(err, entity.ErrCustomerNotFound),
		errors.Is(err, entity.ErrSubscriptionNotFound),
		errors.Is(err, entity.ErrInvoiceNotFound):
		statusCode = http.StatusNotFound
		errorCode = "resource_not_found"
		message = err.Error()

	case errors.Is(err, entity.ErrConflictingData),
		errors.Is(err, entity.ErrInvoiceAlreadyPaid),
		errors.Is(err, entity.ErrSubscriptionCanceled):
		statusCode = http.StatusConflict
		errorCode = "conflict"
		message = err.Error()

	case errors.Is(err, entity.ErrInvalidData),
		errors.Is(err, entity.ErrInvalidPrice):
		statusCode = http.StatusBadRequest
		errorCode = "invalid_request"
		message = err.Error()

	default:
		h.log.LogAttrs(c.Request.Context(), logger.ErrorLevel, "unhandled service error",
			logger.Error(err),
		)
		statusCode = http.StatusInternalServerError
		errorCode = "internal_error"
		message = "An unexpected error occurred"
	}

	h.respondError(c, statusCode, errorCode, message, nil)
}
