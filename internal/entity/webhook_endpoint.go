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

func NewWebhookEndpoint(customerID uuid.UUID, now time.Time, url, secretPrefix, secret string) (*WebhookEndpoint, error) {
	pubID, err := GeneratePublicID("we")
	if err != nil {
		return nil, err
	}
	return &WebhookEndpoint{
		ID:            uuid.New(),
		PublicID:      pubID,
		CustomerID:    customerID,
		URL:           url,
		SecretPrefix:  secretPrefix,
		Secret:        secret,
		EnabledEvents: []string{"*"},
		Enabled:       true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
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

	return fmt.Sprintf("t=%d,v1=%s", timestamp, hex.EncodeToString(mac.Sum(nil)))
}
