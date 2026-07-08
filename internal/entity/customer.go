package entity

import (
	"time"

	"github.com/google/uuid"
)

type Customer struct {
	ID        uuid.UUID
	PublicID  string
	Email     string
	Name      string
	Phone     string
	Metadata  map[string]string
	DeletedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time

	domainEvents DomainEvents
}

func NewCustomer(email, name string, now time.Time) *Customer {
	pubID, _ := GeneratePublicID("cus")
	c := &Customer{
		ID:           uuid.New(),
		PublicID:     pubID,
		Email:        email,
		Name:         name,
		Metadata:     make(map[string]string),
		CreatedAt:    now,
		UpdatedAt:    now,
		domainEvents: make(DomainEvents, 0),
	}

	// Заполняем событие ПОЛНОЙ информацией
	c.domainEvents = append(c.domainEvents, CustomerCreatedEvent{
		CustomerID:    c.ID,
		CustomerPubID: c.PublicID,
		Email:         c.Email,
		Name:          c.Name,
		CreatedAt:     now,
	})

	return c
}

func (c *Customer) GetAndClearEvents() DomainEvents {
	events := c.domainEvents
	c.domainEvents = nil
	return events
}
