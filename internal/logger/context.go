package logger

import (
	"context"

	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"

type contextKey string

const requestIDKey contextKey = "request_id"

func SetRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}
	return ""
}

func GenerateRequestID() string {
	return uuid.New().String()
}
