// Удалить комментарий про generatePublicID
package service

import (
	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"
	"context"
	"crypto/rand"
	"time"
)

const slowOperationThreshold = 5 * time.Second

func logSlowOperation(ctx context.Context, log logger.Logger, op string, start time.Time, attrs ...logger.Attr) {
	duration := time.Since(start)
	if duration <= slowOperationThreshold {
		return
	}

	allAttrs := append([]logger.Attr{
		logger.String("op", op),
		logger.Duration("duration", duration),
	}, attrs...)

	log.LogAttrs(ctx, logger.WarnLevel, "slow operation detected", allAttrs...)
}

func simulatePayment() entity.InvoiceStatus {
	b := make([]byte, 1)
	_, _ = rand.Read(b)

	if b[0] > 25 {
		return entity.InvoiceStatusPaid
	}
	return entity.InvoiceStatusOpen
}
