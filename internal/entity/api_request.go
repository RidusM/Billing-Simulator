package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type APIRequest struct {
	ID             uuid.UUID
	TraceID        uuid.UUID
	Method         string
	Path           string
	QueryParams    json.RawMessage
	RequestBody    json.RawMessage
	Headers        map[string]string
	ResponseStatus int
	ResponseBody   json.RawMessage
	IPAddress      string
	UserAgent      string
	DurationMs     int
	CustomerID     *uuid.UUID
	CreatedAt      time.Time
}
