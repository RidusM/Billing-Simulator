package service

import (
	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"
	"context"
	"math/rand/v2"
	"time"
)

type contextKey string

const (
	ForcePaymentStatusKey contextKey = "force_payment_status"
	PaymentSuccessRateKey contextKey = "payment_success_rate"
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

func simulatePayment(ctx context.Context, rateManager *PaymentRateManager) entity.InvoiceStatus {
	// 1. Проверяем, не форсирован ли статус через контекст (для тестов/демо)
	if status, ok := ctx.Value(ForcePaymentStatusKey).(entity.InvoiceStatus); ok {
		return status
	}

	// 2. Получаем текущий rate из менеджера
	successRate := rateManager.GetSuccessRate()

	// 3. Симулируем платеж
	if rand.Float64() < successRate {
		return entity.InvoiceStatusPaid
	}
	return entity.InvoiceStatusOpen
}

func WithPaymentSuccessRate(ctx context.Context, rate float64) context.Context {
	return context.WithValue(ctx, PaymentSuccessRateKey, rate)
}

func WithForcedPaymentStatus(ctx context.Context, status entity.InvoiceStatus) context.Context {
	return context.WithValue(ctx, ForcePaymentStatusKey, status)
}
