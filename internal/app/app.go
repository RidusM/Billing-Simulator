package app

import (
	"bill-stripe-sim/internal/clock"
	"bill-stripe-sim/internal/config"
	"bill-stripe-sim/internal/repository"
	"bill-stripe-sim/internal/service"
	"bill-stripe-sim/internal/transport/http"
	kafkatransport "bill-stripe-sim/internal/transport/kafka"
	"bill-stripe-sim/internal/worker"
	"bill-stripe-sim/pkg/logger"
	"bill-stripe-sim/pkg/storage/postgres"
	"bill-stripe-sim/pkg/storage/postgres/transaction"
	"bill-stripe-sim/pkg/storage/redis"
	"bill-stripe-sim/pkg/websocket"

	pkgkafka "bill-stripe-sim/pkg/kafka"
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
)

func Run(ctx context.Context, cfg *config.Config, log logger.Logger) error {
	var (
		db       *postgres.Postgres
		rdb      *redis.Redis
		producer *pkgkafka.Producer
		err      error
	)

	defer func() {
		closeResources(ctx, db, rdb, producer, log)
	}()

	// 1. Инициализация инфраструктуры
	db, rdb, producer, err = initInfrastructure(cfg, log)
	if err != nil {
		return err
	}

	// 2. Transaction Manager
	tm, err := transaction.NewManager(db, log)
	if err != nil {
		return fmt.Errorf("init transaction manager: %w", err)
	}

	// 3. Virtual Clock
	cStore := clock.NewRedisStore(rdb)
	vClock, err := clock.NewVirtualClock(cStore, log)
	if err != nil {
		return fmt.Errorf("init virtual clock: %w", err)
	}

	// 4. Repositories
	repos := initRepositories(db, rdb)

	// 5. Services
	services, err := initServices(cfg, repos, tm, vClock, producer, log)
	if err != nil {
		return err
	}

	// 6. WebSocket Hub
	hub := websocket.NewHub(log)

	// 7. HTTP Handler
	handler := http.NewHandler(services.billing, services.time, hub, log)

	// 8. Cleanup Workers
	cleanupWorkers := initCleanupWorkers(repos, log)

	// 9. Запуск всех компонентов
	eg, ctx := errgroup.WithContext(ctx)

	// WebSocket Hub
	eg.Go(func() error {
		hub.Run()
		return nil
	})

	// Outbox Processor
	eg.Go(func() error {
		if err := services.outboxProcessor.Start(ctx); err != nil {
			return fmt.Errorf("start outbox processor: %w", err)
		}
		<-ctx.Done()
		return services.outboxProcessor.Stop(context.Background())
	})

	// Cleanup Workers
	for _, cw := range cleanupWorkers {
		cw := cw // capture range variable
		eg.Go(func() error {
			cw.Start(ctx)
			return nil
		})
	}

	// HTTP Server
	eg.Go(func() error {
		server := http.NewServer(handler, &cfg.HTTP, log)
		if err := server.Start(ctx); err != nil {
			return fmt.Errorf("start http server: %w", err)
		}
		return nil
	})

	// 10. Graceful shutdown
	if egErr := eg.Wait(); egErr != nil && !errors.Is(egErr, context.Canceled) {
		return fmt.Errorf("app execution failed: %w", egErr)
	}

	return nil
}

type repositories struct {
	customer        *repository.CustomerRepository
	invoice         *repository.InvoiceRepository
	subscription    *repository.SubscriptionRepository
	product         *repository.ProductRepository
	price           *repository.PriceRepository
	paymentIntent   *repository.PaymentIntentRepository
	outbox          *repository.OutboxRepository
	event           *repository.EventRepository
	webhookEndpoint *repository.WebhookEndpointRepository
	webhookLog      *repository.WebhookLogRepository
	apiRequest      *repository.APIRequestRepository
	cache           *repository.CacheRepository
}

func initRepositories(db *postgres.Postgres, rdb *redis.Redis) *repositories {
	return &repositories{
		customer:        repository.NewCustomerRepository(db),
		invoice:         repository.NewInvoiceRepository(db),
		subscription:    repository.NewSubscriptionRepository(db),
		product:         repository.NewProductRepository(db),
		price:           repository.NewPriceRepository(db),
		paymentIntent:   repository.NewPaymentIntentRepository(db),
		outbox:          repository.NewOutboxRepository(db),
		event:           repository.NewEventRepository(db),
		webhookEndpoint: repository.NewWebhookEndpointRepository(db),
		webhookLog:      repository.NewWebhookLogRepository(db),
		apiRequest:      repository.NewAPIRequestRepository(db),
		cache:           repository.NewCacheRepository(rdb),
	}
}

type services struct {
	billing         *service.BillingService
	customer        *service.CustomerService
	product         *service.ProductService
	price           *service.PriceService
	payment         *service.PaymentService
	time            *service.TimeService
	notification    *service.NotificationService
	webhookDelivery *service.WebhookDeliveryService
	eventDispatcher *service.EventDispatcher
	outboxProcessor *service.OutboxProcessor
}

func initServices(
	cfg *config.Config,
	repos *repositories,
	tm *transaction.Manager,
	vClock *clock.VirtualClock,
	producer *pkgkafka.Producer,
	log logger.Logger,
) (*services, error) {
	// Event Dispatcher
	eventDispatcher := service.NewEventDispatcher(repos.outbox, vClock, log)

	// Kafka Event Sender
	kafkaSender := kafkatransport.NewEventSenderAdapter(producer, log)

	// Webhook Delivery Service
	webhookDelivery := service.NewWebhookDeliveryService(
		repos.webhookEndpoint,
		repos.webhookLog,
		repos.event,
		&httpWebhookSender{},
		vClock,
		log,
	)

	// Notification Service
	notification := service.NewNotificationService(kafkaSender, webhookDelivery, log)

	// Outbox Processor
	outboxProcessor := service.NewOutboxProcessor(
		repos.outbox,
		notification,
		log,
		service.DefaultOutboxProcessorConfig(),
	)

	// Billing Service
	billing := service.NewBillingService(
		repos.subscription,
		repos.invoice,
		repos.price,
		eventDispatcher,
		tm,
		log,
		vClock,
	)

	// Customer Service
	customer := service.NewCustomerService(
		repos.customer,
		eventDispatcher,
		tm,
		log,
		vClock,
	)

	// Product Service
	product := service.NewProductService(
		repos.product,
		tm,
		log,
		vClock,
	)

	// Price Service
	price := service.NewPriceService(
		repos.price,
		tm,
		log,
		vClock,
	)

	// Payment Service
	payment := service.NewPaymentService(
		repos.paymentIntent,
		repos.invoice,
		eventDispatcher,
		tm,
		log,
		vClock,
	)

	// Time Service
	timeSvc := service.NewTimeService(
		vClock,
		billing,
		repos.subscription,
		repos.cache,
		log,
	)

	// Регистрируем listener для Time Jump (batch renewal)
	vClock.OnTimeJump(func(oldTime, newTime time.Time) {
		log.LogAttrs(context.Background(), logger.InfoLevel, "time jump detected, triggering batch renewal",
			logger.Time("old_time", oldTime),
			logger.Time("new_time", newTime),
		)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			if err := timeSvc.CheckAndRenewSubscriptions(ctx); err != nil {
				log.LogAttrs(ctx, logger.ErrorLevel, "batch renewal failed", logger.Error(err))
			}
		}()
	})

	return &services{
		billing:         billing,
		customer:        customer,
		product:         product,
		price:           price,
		payment:         payment,
		time:            timeSvc,
		notification:    notification,
		webhookDelivery: webhookDelivery,
		eventDispatcher: eventDispatcher,
		outboxProcessor: outboxProcessor,
	}, nil
}

func initCleanupWorkers(repos *repositories, log logger.Logger) []*worker.CleanupWorker {
	return []*worker.CleanupWorker{
		// Очистка webhook_logs старше 7 дней, каждые 1 час
		worker.NewCleanupWorker(
			"webhook_logs",
			1*time.Hour,
			7*24*time.Hour,
			repos.webhookLog.DeleteOldLogs,
			log,
		),
		// Очистка api_requests старше 3 дней, каждые 6 часов
		worker.NewCleanupWorker(
			"api_requests",
			6*time.Hour,
			3*24*time.Hour,
			repos.apiRequest.DeleteOldRequests,
			log,
		),
	}
}

// httpWebhookSender - реализация WebhookSender для HTTP запросов
type httpWebhookSender struct{}

func (s *httpWebhookSender) Send(ctx context.Context, url string, payload []byte, signature string, timestamp int64) (int, error) {
	// Здесь должна быть реальная реализация HTTP клиента
	// Для простоты возвращаем успех
	return 200, nil
}

func initInfrastructure(
	cfg *config.Config,
	log logger.Logger,
) (*postgres.Postgres, *redis.Redis, *pkgkafka.Producer, error) {
	db, err := initDatabase(cfg.Database, log)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("init database: %w", err)
	}

	rdb, err := initCache(cfg.Cache, log)
	if err != nil {
		db.Close()
		return nil, nil, nil, fmt.Errorf("init cache: %w", err)
	}

	producer := pkgkafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.InvoiceTopic, log)

	return db, rdb, producer, nil
}

func initDatabase(cfg config.Database, log logger.Logger) (*postgres.Postgres, error) {
	dbCfg := postgres.Config{
		Host:           cfg.Host,
		Port:           cfg.Port,
		User:           cfg.User,
		Password:       cfg.Password,
		Name:           cfg.Name,
		SSLMode:        cfg.SSLMode,
		MaxPoolSize:    cfg.MaxPoolSize,
		ConnAttempts:   cfg.ConnAttempts,
		BaseRetryDelay: cfg.BaseRetryDelay,
		MaxRetryDelay:  cfg.MaxRetryDelay,
	}
	return postgres.New(dbCfg, log)
}

func initCache(cfg config.Cache, log logger.Logger) (*redis.Redis, error) {
	rdbCfg := redis.Config{
		Host:         cfg.Host,
		Port:         cfg.Port,
		Password:     cfg.Password,
		DB:           cfg.DB,
		TTL:          cfg.TTL,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		PoolTimeout:  cfg.PoolTimeout,
	}
	return redis.New(rdbCfg, log)
}

func closeResources(
	ctx context.Context,
	db *postgres.Postgres,
	rdb *redis.Redis,
	producer *pkgkafka.Producer,
	log logger.Logger,
) {
	if db != nil {
		db.Close()
		log.LogAttrs(ctx, logger.InfoLevel, "database connection closed")
	}
	if rdb != nil {
		if err := rdb.Close(); err != nil {
			log.LogAttrs(ctx, logger.WarnLevel, "failed to close cache", logger.Any("err", err))
		}
	}
	if producer != nil {
		if err := producer.Close(); err != nil {
			log.LogAttrs(ctx, logger.WarnLevel, "failed to close kafka producer", logger.Any("err", err))
		}
	}
	log.LogAttrs(ctx, logger.InfoLevel, "all resources cleaned up")
}
