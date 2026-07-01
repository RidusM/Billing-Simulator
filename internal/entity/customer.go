package entity

import (
	"time"

	"github.com/google/uuid"
)

type Customer struct {
	ID        uuid.UUID
	PublicID  string
	Email     string
	CreatedAt time.Time
}
