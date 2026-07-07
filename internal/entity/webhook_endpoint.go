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
	ID              uuid.UUID
	PublicID        string
	CustomerID      uuid.UUID
	URL             string
	Description     string
	SecretPrefix    string
	SecretEncrypted string
	Secret          string
	EnabledEvents   []string
	Enabled         bool
	DeletedAt       *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func NewWebhookEndpoint(customerID uuid.UUID, url, secretPrefix, secret string) *WebhookEndpoint {
	now := time.Now().UTC()
	return &WebhookEndpoint{
		ID:            uuid.New(),
		PublicID:      GeneratePublicID("we"),
		CustomerID:    customerID,
		URL:           url,
		SecretPrefix:  secretPrefix,
		Secret:        secret,
		EnabledEvents: []string{"*"},
		Enabled:       true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func (e *WebhookEndpoint) ShouldReceiveEvent(eventType string) bool {
	for _, pattern := range e.EnabledEvents {
		if pattern == "*" || pattern == eventType {
			return true
		}
		if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
			prefix := pattern[:len(pattern)-1]
			if len(eventType) >= len(prefix) && eventType[:len(prefix)] == prefix {
				return true
			}
		}
	}
	return false
}

func (e *WebhookEndpoint) SignPayload(payload []byte, timestamp int64) string {
	signedPayload := fmt.Sprintf("%d.%s", timestamp, string(payload))
	mac := hmac.New(sha256.New, []byte(e.Secret))
	mac.Write([]byte(signedPayload))
	return "v1," + hex.EncodeToString(mac.Sum(nil)) // v1, как в Stripe
}
