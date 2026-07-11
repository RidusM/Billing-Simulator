package entity

import (
	"fmt"
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

	AggregateRoot
}

func NewCustomer(email, name string, now time.Time) (*Customer, error) {
	pubID, err := GeneratePublicID("cus")
	if err != nil {
		return nil, fmt.Errorf("failed to generate customer public id: %w", err)
	}

	utc := now.UTC()

	c := &Customer{
		ID:        uuid.New(),
		PublicID:  pubID,
		Email:     email,
		Name:      name,
		Metadata:  NewMetadata(),
		CreatedAt: utc,
		UpdatedAt: utc,
	}

	c.Raise(CustomerCreatedEvent{
		CustomerID:    c.ID,
		CustomerPubID: c.PublicID,
		Email:         c.Email,
		Name:          c.Name,
		CreatedAt:     utc,
	})

	return c, nil
}

func (c *Customer) UpdateEmail(email string, now time.Time) {
	if c.Email == email {
		return // Ничего не менялось
	}
	c.Email = email
	utc := now.UTC()
	c.UpdatedAt = utc

	// ИСПРАВЛЕНО: Теперь бэкенд пользователя мгновенно узнает об обновлении почты!
	c.domainEvents.Raise(CustomerUpdatedEvent{
		CustomerID:    c.ID,
		CustomerPubID: c.PublicID,
		Email:         c.Email,
		Name:          c.Name,
		UpdatedAt:     c.UpdatedAt,
	})
}

func (c *Customer) UpdateName(name string, now time.Time) {
	if c.Name == name {
		return // Ничего не менялось
	}
	c.Name = name
	utc := now.UTC()
	c.UpdatedAt = utc

	// ИСПРАВЛЕНО: Поднимаем событие при смене имени
	c.domainEvents.Raise(CustomerUpdatedEvent{
		CustomerID:    c.ID,
		CustomerPubID: c.PublicID,
		Email:         c.Email,
		Name:          c.Name,
		UpdatedAt:     c.UpdatedAt,
	})
}

func (c *Customer) GetAndClearEvents() DomainEvents {
	return c.domainEvents.ClearAndReturn()
}
