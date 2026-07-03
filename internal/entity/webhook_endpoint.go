package entity

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type WebhookEndpoint struct {
    ID         uuid.UUID
    CustomerID uuid.UUID
    URL        string
    Secret     string
    Enabled    bool
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

func NewWebhookEndpoint(customerID uuid.UUID, url, secret string) *WebhookEndpoint {
    now := time.Now().UTC()
    return &WebhookEndpoint{
        ID:         uuid.New(),
        CustomerID: customerID,
        URL:        url,
        Secret:     secret,
        Enabled:    true,
        CreatedAt:  now,
        UpdatedAt:  now,
    }
}

func (e *WebhookEndpoint) IsActive() bool {
    return e.Enabled && e.URL != ""
}

func (e *WebhookEndpoint) SignPayload(payload []byte, timestamp int64) string {
    signedPayload := fmt.Sprintf("%d.%s", timestamp, string(payload))
    mac := hmac.New(sha256.New, []byte(e.Secret))
    mac.Write([]byte(signedPayload))
    return hex.EncodeToString(mac.Sum(nil))
}