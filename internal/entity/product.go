package entity

import (
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID          uuid.UUID
	PublicID    string
	Name        string
	Description string
	Active      bool
	Metadata    map[string]string
	DeletedAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewProduct(name, description string, now time.Time) *Product {
	pubID, _ := GeneratePublicID("prod")
	return &Product{
		ID:          uuid.New(),
		PublicID:    pubID,
		Name:        name,
		Description: description,
		Active:      true,
		Metadata:    make(map[string]string),
		CreatedAt:   now.UTC(),
		UpdatedAt:   now.UTC(),
	}
}
