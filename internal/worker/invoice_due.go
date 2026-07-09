package worker

import (
	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"
	"context"
	"time"

	"github.com/google/uuid"
)

type InvoiceRepository interface {
	GetOverdue(ctx context.Context, before time.Time, limit uint64) ([]*entity.Invoice, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entity.InvoiceStatus) error
}

type SubscriptionRepository interface {
	GetByInvoiceID(ctx context.Context, invoiceID uuid.UUID) (*entity.Subscription, error)
	Update(ctx context.Context, s *entity.Subscription) error
}

type InvoiceDueWorker struct {
	invoices InvoiceRepository
	subs     SubscriptionRepository
	clock    VirtualClock
	log      logger.Logger
	cfg      InvoiceDueConfig
	done     chan struct{}
}

type InvoiceDueConfig struct {
	PollInterval time.Duration
	BatchSize    uint64
}

func DefaultInvoiceDueConfig() InvoiceDueConfig {
	return InvoiceDueConfig{
		PollInterval: 5 * time.Minute, // Проверяем каждые 5 минут
		BatchSize:    100,
	}
}

func NewInvoiceDueWorker(
	invoices InvoiceRepository,
	subs SubscriptionRepository,
	clock VirtualClock,
	log logger.Logger,
	cfg InvoiceDueConfig,
) *InvoiceDueWorker {
	return &InvoiceDueWorker{
		invoices: invoices,
		subs:     subs,
		clock:    clock,
		log:      log,
		cfg:      cfg,
		done:     make(chan struct{}),
	}
}

func (w *InvoiceDueWorker) Start(ctx context.Context) {
	w.log.Info("starting invoice due worker",
		"poll_interval", w.cfg.PollInterval,
	)

	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("invoice due worker stopped")
			return
		case <-w.done:
			return
		case <-ticker.C:
			w.processOverdueInvoices(ctx)
		}
	}
}

func (w *InvoiceDueWorker) Stop() {
	close(w.done)
}

func (w *InvoiceDueWorker) processOverdueInvoices(ctx context.Context) {
	now := w.clock.Now()
	invoices, err := w.invoices.GetOverdue(ctx, now, w.cfg.BatchSize)
	if err != nil {
		w.log.Error("failed to get overdue invoices", "error", err)
		return
	}

	if len(invoices) == 0 {
		return
	}

	w.log.Info("processing overdue invoices", "count", len(invoices))

	for _, inv := range invoices {
		w.processOverdueInvoice(ctx, inv)
	}
}

func (w *InvoiceDueWorker) processOverdueInvoice(ctx context.Context, inv *entity.Invoice) {
	now := w.clock.Now()

	// Меняем статус инвойса (если нужно)
	// Для простоты оставляем статус open, но можно добавить поле is_overdue
	// Или менять на uncollectible, если прошло много времени

	// Если инвойс привязан к подписке, меняем статус подписки на past_due
	if inv.SubscriptionID != nil {
		sub, err := w.subs.GetByInvoiceID(ctx, *inv.SubscriptionID)
		if err != nil {
			w.log.Error("failed to get subscription for overdue invoice",
				"invoice_id", inv.ID,
				"error", err,
			)
			return
		}

		if sub != nil && sub.Status == entity.SubscriptionStatusActive {
			sub.MarkPastDue(now)
			if err := w.subs.Update(ctx, sub); err != nil {
				w.log.Error("failed to mark subscription as past_due",
					"subscription_id", sub.ID,
					"error", err,
				)
			} else {
				w.log.Info("subscription marked as past_due due to overdue invoice",
					"subscription_id", sub.ID,
					"invoice_id", inv.ID,
				)
			}
		}
	}

	w.log.Debug("processed overdue invoice",
		"invoice_id", inv.ID,
		"customer_id", inv.CustomerID,
		"amount", inv.Amount,
	)
}
