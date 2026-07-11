package app

import (
	"context"
	"fmt"
	"time"

	"bill-stripe-sim/internal/clock"
	"bill-stripe-sim/internal/config"
	"bill-stripe-sim/internal/repository"
	"bill-stripe-sim/internal/service"
	"bill-stripe-sim/internal/worker"
	"bill-stripe-sim/pkg/kafka"
	"bill-stripe-sim/pkg/kafka/dlq"
	"bill-stripe-sim/pkg/logger"
	"bill-stripe-sim/pkg/storage/postgres"
	"bill-stripe-sim/pkg/storage/postgres/transaction"
	"bill-stripe-sim/pkg/storage/redis"
	"bill-stripe-sim/pkg/websocket"
	handler "bill-stripe-sim/transport/http"
	"bill-stripe-sim/transport/http/webhooksender"
	kafkatransport "bill-stripe-sim/transport/kafka"
	wstransport "bill-stripe-sim/transport/websocket"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

// App — держит всё, что нужно грациозно остановить при shutdown.
type App struct {
	log logger.Logger

	httpServer      *handler.Server
	outboxProcessor *worker.OutboxProcessor
	invoiceDue      *worker.InvoiceDueWorker
	renewalWorker   *worker.SubscriptionRenewalWorker
	webhookRetry    *worker.WebhookRetryWorker
	cleanupWorkers  []*worker.CleanupWorker
	kafkaProducer   *kafka.Producer
	dlq             *dlq.DLQ
	hub             *websocket.Hub
}

// New — единственное место в проекте, где создаются конкретные типы и связываются интерфейсами.
// ВСЕ конструкторы принимают интерфейсы, поэтому здесь единственное место, знающее про
// конкретные *postgres.Postgres/*redis.Redis/*kafka.Producer и т.д.
//
// Некоторые вызовы (postgres.New/redis.New/transaction.NewManager) — под вашу реальную
// сигнатуру pkg/storage; здесь показан ожидаемый контракт, поправьте имена под факт.
func New(ctx context.Context, cfg *config.Config, log logger.Logger) (*App, error) {
	const op = "app.New"

	// ---------- Инфраструктура ----------

	pg, err := postgres.New(ctx, cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("%s: connect postgres: %w", op, err)
	}

	rdb, err := redis.New(ctx, cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("%s: connect redis: %w", op, err)
	}

	tm := transaction.NewManager(pg)

	clockStore := clock.NewRedisStore(rdb)
	vClock, err := clock.NewVirtualClock(clockStore, log)
	if err != nil {
		return nil, fmt.Errorf("%s: init virtual clock: %w", op, err)
	}

	// ---------- WebSocket Hub (дашборд) ----------

	hub := websocket.NewHub(log)
	go hub.Run()
	broadcaster := wstransport.NewHubBroadcaster(hub)

	// ---------- Kafka producer + DLQ + адаптер под service.EventSender ----------

	kafkaProducer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.EventsTopic, log)
	kafkaDLQ := dlq.New(kafkaProducer, cfg.Kafka.DLQTopic, log)
	eventSender := kafkatransport.NewEventSenderAdapter(kafkaProducer, log)

	// ---------- Репозитории ----------

	customerRepo := repository.NewCustomerRepository(pg)
	productRepo := repository.NewProductRepository(pg)
	priceRepo := repository.NewPriceRepository(pg)
	subscriptionRepo := repository.NewSubscriptionRepository(pg)
	invoiceRepo := repository.NewInvoiceRepository(pg)
	paymentIntentRepo := repository.NewPaymentIntentRepository(pg)
	outboxRepo := repository.NewOutboxRepository(pg)
	webhookEndpointRepo := repository.NewWebhookEndpointRepository(pg)
	webhookLogRepo := repository.NewWebhookLogRepository(pg)
	eventRepo := repository.NewEventRepository(pg)
	cacheRepo := repository.NewCacheRepository(rdb)

	// ---------- Сервисы ----------

	dispatcher := service.NewEventDispatcher(outboxRepo, vClock, log)

	customerSvc := service.NewCustomerService(customerRepo, dispatcher, tm, log, vClock)
	productSvc := service.NewProductService(productRepo, tm, log, vClock)
	priceSvc := service.NewPriceService(priceRepo, tm, log, vClock)
	rateManager := service.NewPaymentRateManager(cfg.Simulation.DefaultPaymentSuccessRate)

	billingSvc := service.NewBillingService(subscriptionRepo, invoiceRepo, priceRepo, dispatcher, tm, log, vClock, rateManager)
	paymentSvc := service.NewPaymentService(paymentIntentRepo, invoiceRepo, dispatcher, tm, log, vClock, rateManager)
	invoiceQuerySvc := service.NewInvoiceQueryService(invoiceRepo)
	webhookEndpointSvc := service.NewWebhookEndpointService(webhookEndpointRepo, tm, vClock)

	webhookSender := webhooksender.NewHTTPSender()
	webhookDeliverySvc := service.NewWebhookDeliveryService(webhookEndpointRepo, webhookLogRepo, eventRepo, webhookSender, vClock, log)

	notificationSvc := service.NewNotificationService(eventSender, webhookDeliverySvc, broadcaster, log)

	timeSvc := service.NewTimeService(vClock, billingSvc, subscriptionRepo, cacheRepo, log)

	// ---------- Воркеры ----------

	outboxProcessor := worker.NewOutboxProcessor(outboxRepo, notificationSvc, log, worker.DefaultOutboxProcessorConfig())
	invoiceDueWorker := worker.NewInvoiceDueWorker(invoiceRepo, subscriptionRepo, vClock, log, worker.DefaultInvoiceDueConfig())
	renewalWorker := worker.NewSubscriptionRenewalWorker(subscriptionRepo, billingSvc, vClock, log, worker.DefaultSubscriptionRenewalConfig())
	webhookRetryWorker := worker.NewWebhookRetryWorker(webhookLogRepo, webhookEndpointRepo, webhookSender, vClock, log, worker.DefaultWebhookRetryConfig())

	outboxCleanup := worker.NewCleanupWorker("outbox_events", 1*time.Hour, 24*time.Hour, outboxRepo.DeleteOldProcessed, log)

	// ---------- HTTP ----------

	facade := handler.NewBillingFacade(customerSvc, billingSvc, timeSvc, rateManager, priceSvc)
	billingHandler := handler.NewHandler(facade, log)

	resourceHandlers := handler.NewResourceHandlers(productSvc, priceSvc, invoiceQuerySvc, paymentSvc, webhookEndpointSvc, customerSvc, log)
	v1 := billingHandler.Engine().Group("/v1")
	resourceHandlers.RegisterRoutes(v1)

	dashboardHandler := handler.NewDashboardHandler(hub, log)
	billingHandler.Engine().GET("/ws", func(c *gin.Context) {
		dashboardHandler.ServeWS(c.Writer, c.Request)
	})

	httpServer := handler.NewServer(billingHandler, cfg.HTTP, log)

	return &App{
		log:             log,
		httpServer:      httpServer,
		outboxProcessor: outboxProcessor,
		invoiceDue:      invoiceDueWorker,
		renewalWorker:   renewalWorker,
		webhookRetry:    webhookRetryWorker,
		cleanupWorkers:  []*worker.CleanupWorker{outboxCleanup},
		kafkaProducer:   kafkaProducer,
		dlq:             kafkaDLQ,
		hub:             hub,
	}, nil
}

// Run — запускает всё и блокируется до отмены ctx (например, по SIGTERM в main.go).
func (a *App) Run(ctx context.Context) error {
	const op = "app.Run"

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error { return a.httpServer.Start(ctx) })
	eg.Go(func() error { return a.outboxProcessor.Start(ctx) })

	eg.Go(func() error { a.invoiceDue.Start(ctx); return nil })
	eg.Go(func() error { a.renewalWorker.Start(ctx); return nil })
	eg.Go(func() error { a.webhookRetry.Start(ctx); return nil })

	for _, w := range a.cleanupWorkers {
		w := w
		eg.Go(func() error { w.Start(ctx); return nil })
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// Shutdown — грациозно останавливает воркеры и HTTP-сервер в правильном порядке:
// сначала перестаём принимать новый трафик (HTTP), затем даём воркерам дообработать
// то, что уже в полёте, и только потом закрываем Kafka/DLQ.
func (a *App) Shutdown(ctx context.Context) error {
	_ = a.httpServer.Stop(ctx)

	_ = a.outboxProcessor.Stop(ctx)
	a.invoiceDue.Stop()
	a.renewalWorker.Stop()
	a.webhookRetry.Stop()
	for _, w := range a.cleanupWorkers {
		w.Stop()
	}

	_ = a.dlq.Close()
	_ = a.kafkaProducer.Close()

	return nil
}
