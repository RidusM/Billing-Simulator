package http

import (
	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

const _maxRequestBodySize = 1 << 20

type BillingService interface {
	CreateCustomer(ctx context.Context, email string) (*entity.Customer, error)
	CreateSubscription(ctx context.Context, customerID uuid.UUID, priceID string) (*entity.Subscription, error)
	CancelSubscription(ctx context.Context, subID string) error
	GetSubscription(ctx context.Context, subID uuid.UUID) (*entity.Subscription, error)
}

type BillingHandler struct {
	svc    BillingService
	log    logger.Logger
	router *gin.Engine
}

func NewHandler(svc BillingService, log logger.Logger) *BillingHandler {
	h := &BillingHandler{
		svc: svc,
		log: log,
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, _maxRequestBodySize)
		c.Next()
	})

	router.Use(otelgin.Middleware("billing-service"))
	router.Use(h.requestIDMiddleware())
	router.Use(h.loggingMiddleware())
	router.Use(gin.Recovery())

	h.router = router
	h.setupRoutes()

	return h
}

func (h *BillingHandler) Engine() *gin.Engine {
	return h.router
}
