package http

import (
	"net/http"

	_ "bill-stripe-sim/docs" // required for Swagger

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Billing Stripe Simulation API
// @version 1.0
// @description API for simulating stripe billing operations
// @termsOfService httpL//swagger.io/terms
// @contact.name RidusM
// @contact.email esandalov04@gmail.com
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0
// @host localhost:8080
// @BasePath /
func (h *BillingHandler) setupRoutes() {
	h.router.GET("/healthz", h.Health)

	v1 := h.router.Group("/v1")
	v1.Use()
	{
		v1.POST("/customers", h.CreateCustomer)
		subs := h.router.Group("/subscriptions")
		subs.Use()
		{
			subs.POST("/", h.CreateSubscription)
			subs.GET("/:id", h.GetSubscription)
		}
	}

	h.router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{})
	})
	h.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
