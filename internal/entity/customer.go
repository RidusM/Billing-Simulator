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
		CreatedAt:    now.UTC(),
		UpdatedAt:    now.UTC(),
		domainEvents: make(DomainEvents, 0),
	}

	c.domainEvents.Raise(CustomerCreatedEvent{
		CustomerID:    c.ID,
		CustomerPubID: c.PublicID,
		Email:         c.Email,
		Name:          c.Name,
		CreatedAt:     now.UTC(),
	})

	return c
}

func (c *Customer) UpdateEmail(email string, now time.Time) {
	c.Email = email
	c.UpdatedAt = now.UTC()
}

func (c *Customer) UpdateName(name string, now time.Time) {
	c.Name = name
	c.UpdatedAt = now.UTC()
}

func (c *Customer) GetAndClearEvents() DomainEvents {
	return c.domainEvents.ClearAndReturn()
}
