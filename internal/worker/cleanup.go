package worker

import (
	"bill-stripe-sim/pkg/logger"
	"context"
	"time"
)

type CleanupFunc func(ctx context.Context, olderThan time.Duration) (int64, error)

type CleanupWorker struct {
	name      string
	interval  time.Duration
	retention time.Duration
	cleanupFn CleanupFunc
	log       logger.Logger
	stopCh    chan struct{}
}

func NewCleanupWorker(
	name string,
	interval, retention time.Duration,
	cleanupFn CleanupFunc,
	log logger.Logger,
) *CleanupWorker {
	return &CleanupWorker{
		name:      name,
		interval:  interval,
		retention: retention,
		cleanupFn: cleanupFn,
		log:       log,
		stopCh:    make(chan struct{}),
	}
}

func (w *CleanupWorker) Start(ctx context.Context) {
	w.log.Info("starting cleanup worker", "name", w.name, "interval", w.interval, "retention", w.retention)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("cleanup worker stopped", "name", w.name)
			return
		case <-w.stopCh:
			w.log.Info("cleanup worker stopped by signal", "name", w.name)
			return
		case <-ticker.C:
			count, err := w.cleanupFn(ctx, w.retention)
			if err != nil {
				w.log.Error("cleanup failed", "name", w.name, "error", err)
			} else if count > 0 {
				w.log.Info("cleanup completed", "name", w.name, "deleted_count", count)
			}
		}
	}
}

func (w *CleanupWorker) Stop() {
	close(w.stopCh)
}
