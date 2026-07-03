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

func NewCustomer(email string) *Customer {
	return &Customer{
		ID: uuid.New(),
		PublicID: generatePublicID("cus"),
		Email: email,
		CreatedAt: time.Now().UTC(),
	}
}

func generatePublicID(prefix string) string{
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}
