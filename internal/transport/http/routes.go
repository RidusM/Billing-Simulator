package http

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
}
