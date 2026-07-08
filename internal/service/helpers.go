package service

import (
	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"
	"context"
	"math/rand/v2"
	"time"
)

type contextKey string

const ForcePaymentStatusKey contextKey = "force_payment_status"
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

func simulatePayment(ctx context.Context, successRate float64) entity.InvoiceStatus {
	// 1. Проверяем, не форсирован ли статус через контекст (для тестов/демо)
	if status, ok := ctx.Value(ForcePaymentStatusKey).(entity.InvoiceStatus); ok {
		return status
	}

	// 2. Используем современный math/rand/v2 с плавающей точкой (например, 0.75 для 75% успеха)
	if rand.Float64() < successRate {
		return entity.InvoiceStatusPaid
	}

	return entity.InvoiceStatusOpen
}
