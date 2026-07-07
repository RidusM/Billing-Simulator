package service

import (
	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type OutboxRepository interface {
	GetUnprocessed(ctx context.Context, limit int, olderThan time.Duration) ([]*entity.OutboxEvent, error)
	MarkProcessed(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, errorMsg string) error
	DeleteOldProcessed(ctx context.Context, olderThan time.Duration) (int64, error)
}

// OutboxProcessor — фоновый процессор outbox событий
type OutboxProcessor struct {
	outbox       OutboxRepository
	notification *NotificationService
	log          logger.Logger

	cfg OutboxProcessorConfig

	// Для graceful shutdown
	wg   sync.WaitGroup
	done chan struct{}
}

type OutboxProcessorConfig struct {
	PollInterval    time.Duration // Как часто проверять outbox (default: 1s)
	BatchSize       int           // Сколько событий обрабатывать за раз (default: 100)
	MinAge          time.Duration // Минимальный возраст события перед обработкой (default: 0)
	MaxRetries      int           // Максимум попыток обработки (default: 5)
	WorkerCount     int           // Количество воркеров (default: 5)
	CleanupInterval time.Duration // Как часто чистить старые события (default: 1h)
	RetentionPeriod time.Duration // Сколько хранить обработанные события (default: 24h)
}

func DefaultOutboxProcessorConfig() OutboxProcessorConfig {
	return OutboxProcessorConfig{
		PollInterval:    1 * time.Second,
		BatchSize:       100,
		MinAge:          0,
		MaxRetries:      5,
		WorkerCount:     5,
		CleanupInterval: 1 * time.Hour,
		RetentionPeriod: 24 * time.Hour,
	}
}

func NewOutboxProcessor(
	outbox OutboxRepository,
	notification *NotificationService,
	log logger.Logger,
	cfg OutboxProcessorConfig,
) *OutboxProcessor {
	return &OutboxProcessor{
		outbox:       outbox,
		notification: notification,
		log:          log,
		cfg:          cfg,
		done:         make(chan struct{}),
	}
}

// Start — запускает процессор
func (p *OutboxProcessor) Start(ctx context.Context) error {
	p.log.Info("starting outbox processor",
		"poll_interval", p.cfg.PollInterval,
		"batch_size", p.cfg.BatchSize,
		"worker_count", p.cfg.WorkerCount,
	)

	// Запускаем воркеров
	for i := 0; i < p.cfg.WorkerCount; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}

	// Запускаем cleanup воркер
	p.wg.Add(1)
	go p.cleanupWorker(ctx)

	return nil
}

// Stop — останавливает процессор
func (p *OutboxProcessor) Stop(ctx context.Context) error {
	p.log.Info("stopping outbox processor")

	close(p.done)

	// Ждём завершения воркеров с таймаутом
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.log.Info("outbox processor stopped gracefully")
		return nil
	case <-ctx.Done():
		p.log.Warn("outbox processor stopped with timeout")
		return ctx.Err()
	}
}

func (p *OutboxProcessor) worker(ctx context.Context, id int) {
	defer p.wg.Done()

	p.log.Debug("outbox worker started", "worker_id", id)

	ticker := time.NewTicker(p.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.done:
			return
		case <-ticker.C:
			p.processBatch(ctx, id)
		}
	}
}

func (p *OutboxProcessor) processBatch(ctx context.Context, workerID int) {
	events, err := p.outbox.GetUnprocessed(ctx, p.cfg.BatchSize, p.cfg.MinAge)
	if err != nil {
		p.log.Error("failed to get unprocessed events",
			"worker_id", workerID,
			"error", err,
		)
		return
	}

	if len(events) == 0 {
		return
	}

	p.log.Debug("processing batch",
		"worker_id", workerID,
		"count", len(events),
	)

	for _, event := range events {
		select {
		case <-ctx.Done():
			return
		default:
			p.processEvent(ctx, event)
		}
	}
}

func (p *OutboxProcessor) processEvent(ctx context.Context, event *entity.OutboxEvent) {
	// Извлекаем customerID из payload (нужно десериализовать для этого)
	customerID, err := p.extractCustomerID(event)
	if err != nil {
		p.log.Error("failed to extract customerID",
			"event_id", event.ID,
			"error", err,
		)
		p.markFailed(ctx, event, fmt.Sprintf("extract customerID: %v", err))
		return
	}

	// Передаём УЖЕ СЕРИАЛИЗОВАННЫЙ payload
	if err := p.notification.HandleDomainEvent(
		ctx,
		event.EventType,
		customerID,
		event.Payload, // ← Передаём []byte напрямую!
	); err != nil {
		p.log.Error("failed to handle domain event",
			"event_id", event.ID,
			"event_type", event.EventType,
			"error", err,
		)
		p.markFailed(ctx, event, fmt.Sprintf("handle: %v", err))
		return
	}

	if err := p.outbox.MarkProcessed(ctx, event.ID); err != nil {
		p.log.Error("failed to mark event as processed",
			"event_id", event.ID,
			"error", err,
		)
	}
}

func (p *OutboxProcessor) extractCustomerID(event *entity.OutboxEvent) (uuid.UUID, error) {
	// Десериализуем только для извлечения customerID
	var data struct {
		CustomerID uuid.UUID `json:"customer_id"`
	}
	if err := json.Unmarshal(event.Payload, &data); err != nil {
		return uuid.Nil, err
	}
	return data.CustomerID, nil
}

func (p *OutboxProcessor) markProcessed(ctx context.Context, event *entity.OutboxEvent) {
	if err := p.outbox.MarkProcessed(ctx, event.ID); err != nil {
		p.log.Error("failed to mark event as processed",
			"event_id", event.ID,
			"error", err,
		)
	}
}

func (p *OutboxProcessor) markFailed(ctx context.Context, event *entity.OutboxEvent, errMsg string) {
	if err := p.outbox.MarkFailed(ctx, event.ID, errMsg); err != nil {
		p.log.Error("failed to mark event as failed",
			"event_id", event.ID,
			"error", err,
		)
	}
}

func (p *OutboxProcessor) cleanupWorker(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.cfg.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.done:
			return
		case <-ticker.C:
			count, err := p.outbox.DeleteOldProcessed(ctx, p.cfg.RetentionPeriod)
			if err != nil {
				p.log.Error("failed to cleanup old events", "error", err)
			} else if count > 0 {
				p.log.Debug("cleaned up old events", "count", count)
			}
		}
	}
}
