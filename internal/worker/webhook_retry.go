package worker

import (
	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type WebhookLogRepository interface {
	GetPendingForRetry(ctx context.Context, currentTime time.Time, limit int) ([]*entity.WebhookLog, error)
	Update(ctx context.Context, wl *entity.WebhookLog) error
	MarkFailed(ctx context.Context, id uuid.UUID, errorMessage string, nextAttemptAt time.Time, now time.Time) error
}

type WebhookSender interface {
	Send(ctx context.Context, url string, payload []byte, signature string, timestamp int64) (int, error)
}

type WebhookEndpointRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entity.WebhookEndpoint, error)
}

type WebhookRetryWorker struct {
	logs      WebhookLogRepository
	endpoints WebhookEndpointRepository
	sender    WebhookSender
	clock     VirtualClock
	log       logger.Logger
	cfg       WebhookRetryConfig
	done      chan struct{}
}

type WebhookRetryConfig struct {
	PollInterval time.Duration
	BatchSize    int
	MaxAttempts  int
}

func DefaultWebhookRetryConfig() WebhookRetryConfig {
	return WebhookRetryConfig{
		PollInterval: 1 * time.Second,
		BatchSize:    50,
		MaxAttempts:  5,
	}
}

func NewWebhookRetryWorker(
	logs WebhookLogRepository,
	endpoints WebhookEndpointRepository,
	sender WebhookSender,
	clock VirtualClock,
	log logger.Logger,
	cfg WebhookRetryConfig,
) *WebhookRetryWorker {
	return &WebhookRetryWorker{
		logs:      logs,
		endpoints: endpoints,
		sender:    sender,
		clock:     clock,
		log:       log,
		cfg:       cfg,
		done:      make(chan struct{}),
	}
}

func (w *WebhookRetryWorker) Start(ctx context.Context) {
	w.log.Info("starting webhook retry worker",
		"poll_interval", w.cfg.PollInterval,
		"batch_size", w.cfg.BatchSize,
	)

	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("webhook retry worker stopped")
			return
		case <-w.done:
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *WebhookRetryWorker) Stop() {
	close(w.done)
}

func (w *WebhookRetryWorker) processBatch(ctx context.Context) {
	now := w.clock.Now()
	logs, err := w.logs.GetPendingForRetry(ctx, now, w.cfg.BatchSize)
	if err != nil {
		w.log.Error("failed to get pending webhooks", "error", err)
		return
	}

	if len(logs) == 0 {
		return
	}

	w.log.Debug("processing webhook retries", "count", len(logs))

	for _, wl := range logs {
		w.retryWebhook(ctx, wl)
	}
}

func (w *WebhookRetryWorker) retryWebhook(ctx context.Context, wl *entity.WebhookLog) {
	ep, err := w.endpoints.GetByID(ctx, wl.EndpointID)
	if err != nil {
		w.log.Error("failed to get webhook endpoint",
			"webhook_log_id", wl.ID,
			"endpoint_id", wl.EndpointID,
			"error", err,
		)
		return
	}

	now := w.clock.Now()
	timestamp := now.Unix()
	signature := ep.SignPayload(wl.Payload, timestamp)

	statusCode, err := w.sender.Send(ctx, wl.TargetURL, wl.Payload, signature, timestamp)
	if err == nil && statusCode >= 200 && statusCode < 300 {
		// Успех
		wl.MarkDelivered(statusCode, now)
		if err := w.logs.Update(ctx, wl); err != nil {
			w.log.Error("failed to mark webhook as delivered",
				"webhook_log_id", wl.ID,
				"error", err,
			)
		}
		w.log.Info("webhook delivered successfully",
			"webhook_log_id", wl.ID,
			"target_url", wl.TargetURL,
			"status_code", statusCode,
		)
		return
	}

	// Неудача — планируем следующий retry
	if wl.Attempt >= w.cfg.MaxAttempts {
		w.log.Warn("webhook max attempts reached, marking as failed",
			"webhook_log_id", wl.ID,
			"attempt", wl.Attempt,
		)
		return
	}

	// Экспоненциальный бэкофф: 1s, 2s, 4s, 8s, 16s
	backoffSeconds := int64(1 << (wl.Attempt - 1))
	nextAttemptAt := now.Add(time.Duration(backoffSeconds) * time.Second)

	errMsg := "request failed"
	if err != nil {
		errMsg = err.Error()
	} else {
		errMsg = fmt.Sprintf("HTTP %d", statusCode)
	}

	if err := w.logs.MarkFailed(ctx, wl.ID, errMsg, nextAttemptAt, now); err != nil {
		w.log.Error("failed to mark webhook as failed",
			"webhook_log_id", wl.ID,
			"error", err,
		)
	}

	w.log.Debug("webhook retry scheduled",
		"webhook_log_id", wl.ID,
		"attempt", wl.Attempt+1,
		"next_attempt_at", nextAttemptAt,
	)
}
