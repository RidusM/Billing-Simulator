package entity

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

type Customer struct {
	ID        uuid.UUID
	PublicID  string
	Email     string
	CreatedAt time.Time
}

func NewCustomer(email string, now time.Time) *Customer {
	return &Customer{
		ID:        uuid.New(),
		PublicID:  generatePublicID("cus"),
		Email:     email,
		CreatedAt: now.UTC(),
	}
}

func generatePublicID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}
