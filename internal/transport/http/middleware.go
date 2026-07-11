package handler

import (
	"bill-stripe-sim/pkg/logger"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	_statusServerError = 500
	_statusClientError = 400
)

func (h *BillingHandler) requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(logger.RequestIDHeader)
		if reqID == "" {
			reqID = logger.GenerateRequestID()
		}
		ctx := logger.SetRequestID(c.Request.Context(), reqID)
		c.Request = c.Request.WithContext(ctx)
		c.Header("X-Request-ID", reqID)

		c.Next()
	}
}

func (h *BillingHandler) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()
		ctx := c.Request.Context()

		level := logger.InfoLevel
		if status >= _statusServerError {
			level = logger.ErrorLevel
		} else if status >= _statusClientError {
			level = logger.WarnLevel
		}

		h.log.LogAttrs(ctx, level, "http request",
			logger.String("method", c.Request.Method),
			logger.String("path", c.FullPath()),
			logger.Int("status", status),
			logger.Duration("duration", duration),
			logger.String("client_ip", c.ClientIP()),
		)
	}
}
