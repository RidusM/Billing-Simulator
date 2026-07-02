package service

import (
	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/brianvoe/gofakeit/v7"
)

const _slowOperationThreshold = 5 * time.Second

func logSlowOperation(
	ctx context.Context,
	log logger.Logger,
	op string,
	start time.Time,
	attrs ...logger.Attr,
) {
	duration := time.Since(start)
	if duration <= _slowOperationThreshold {
		return
	}

	allAttrs := append([]logger.Attr{
		logger.String("op", op),
		logger.Duration("duration", duration),
	}, attrs...)
	log.LogAttrs(ctx, logger.WarnLevel, "slow operation detected", allAttrs...)
}

func generatePublicID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b))
}

func calculateNextPeriod(currentEnd time.Time) (start, end time.Time) {
	start = currentEnd
	end = currentEnd.AddDate(0, 1, 0)
	return
}

func getRandomAmount() int64 {
	return int64(gofakeit.Number(500, 10000))
}

func getRandomCurrency() string {
	return gofakeit.CurrencyShort()
}

func simulatePayment() entity.InvoiceStatus {
	b := make([]byte, 1)
	_, _ = rand.Read(b)
	if b[0] > 12 {
		return entity.InvoiceStatusPaid
	}
	return entity.InvoiceStatusOpen
}
