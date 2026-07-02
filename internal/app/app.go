package app

import (
	"bill-stripe-sim/internal/clock"
	"bill-stripe-sim/internal/config"
	"bill-stripe-sim/internal/repository"
	"bill-stripe-sim/internal/service"
	"bill-stripe-sim/internal/transport/http"
	"bill-stripe-sim/internal/transport/kafka"
	"bill-stripe-sim/pkg/logger"
	"bill-stripe-sim/pkg/storage/postgres"
	"bill-stripe-sim/pkg/storage/postgres/transaction"
	"bill-stripe-sim/pkg/storage/redis"

	pkgkafka "bill-stripe-sim/pkg/kafka"
	"context"
	"errors"
	"fmt"

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

	db, rdb, producer, err = initInfrastructure(cfg, log)
	if err != nil {
		return err
	}

	tm, err := transaction.NewManager(db, log)
	if err != nil {
		return fmt.Errorf("init transaction manager: %w", err)
	}

	handler, err := initServices(cfg, db, tm, rdb, producer, log)
	if err != nil {
		return err
	}

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		server := http.NewServer(*handler, &cfg.HTTP, log)
		if err := server.Start(ctx); err != nil {
			return fmt.Errorf("start http server: %w", err)
		}
		return nil
	})

	if egErr := eg.Wait(); egErr != nil && !errors.Is(egErr, context.Canceled) {
		return fmt.Errorf("app execution failed: %w", egErr)
	}

	return nil
}

func initServices(
	cfg *config.Config,
	db *postgres.Postgres,
	tm *transaction.Manager,
	rdb *redis.Redis,
	producer *pkgkafka.Producer,
	log logger.Logger,
) (*http.BillingHandler, error) {
	cStore := clock.NewRedisStore(rdb)
	vClock := clock.NewVirtualClock(cStore)

	cusRepo := repository.NewCustomerRepository(db)
	invRepo := repository.NewInvoiceRepository(db)
	subRepo := repository.NewSubscriptionRepository(db)
	cacheRepo := repository.NewCacheRepository(rdb)

	eventSender := kafka.NewEventSender(producer, cfg.Kafka.InvoiceTopic, cfg.Kafka.SubscriptionTopic)

	notifySvc := service.NewNotificationService(eventSender)
	billSvc := service.NewBillingService(
		cusRepo,
		invRepo,
		subRepo,
		cacheRepo,
		tm,
		log,
		vClock,
		notifySvc,
	)

	handler := http.NewHandler(billSvc, log)

	return handler, nil
}

func initInfrastructure(
	cfg *config.Config,
	log logger.Logger,
) (*postgres.Postgres, *redis.Redis, *pkgkafka.Producer, error) {
	db, err := initDatabase(cfg.Database, log)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("init database: %w", err)
	}

	rdb, err := initCache(cfg.Cache)
	if err != nil {
		db.Close()
		return nil, nil, nil, fmt.Errorf("init cache: %w", err)
	}

	producer := pkgkafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.InvoiceTopic, log)

	return db, rdb, producer, nil
}

func initDatabase(cfg config.Database, log logger.Logger) (*postgres.Postgres, error) {
	dbCfg := postgres.Config{
		Host:     cfg.Host,
		Port:     cfg.Port,
		User:     cfg.User,
		Password: cfg.Password,
		Name:     cfg.Name,
		SSLMode:  cfg.SSLMode,
	}
	return postgres.New(dbCfg, log,
		postgres.WithMaxPoolSize(cfg.MaxPoolSize),
		postgres.WithConnAttempts(cfg.ConnAttempts),
		postgres.WithBaseRetryDelay(cfg.BaseRetryDelay),
		postgres.WithMaxRetryDelay(cfg.MaxRetryDelay),
	)
}

func initCache(cfg config.Cache) (*redis.Redis, error) {
	rdbCfg := redis.Config{
		Host:     cfg.Host,
		Port:     cfg.Port,
		Password: cfg.Password,
		DB:       cfg.DB,
		TTL:      cfg.TTL,
	}
	return redis.New(rdbCfg,
		redis.WithMinIdleConns(cfg.MinIdleConns),
		redis.WithPoolSize(cfg.PoolSize),
		redis.WithPoolTimeout(cfg.PoolTimeout),
		redis.WithTTL(cfg.TTL),
	)
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
