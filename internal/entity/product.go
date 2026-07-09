package entity

import (
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID           uuid.UUID
	PublicID     string
	Name         string
	Description  string
	Active       bool
	Metadata     map[string]string
	DeletedAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	domainEvents DomainEvents
}

func NewProduct(name, description string, now time.Time) *Product {
	pubID, _ := GeneratePublicID("prod")
	p := &Product{
		ID:           uuid.New(),
		PublicID:     pubID,
		Name:         name,
		Description:  description,
		Active:       true,
		Metadata:     make(map[string]string),
		CreatedAt:    now.UTC(),
		UpdatedAt:    now.UTC(),
		domainEvents: make(DomainEvents, 0),
	}

	p.domainEvents = append(p.domainEvents, ProductCreatedEvent{
		ProductID:    p.ID,
		ProductPubID: p.PublicID,
		Name:         p.Name,
		Description:  p.Description,
		CreatedAt:    now,
	})

	return p
}

func (p *Product) Update(name, description string, active bool, now time.Time) {
	p.Name = name
	p.Description = description
	p.Active = active
	p.UpdatedAt = now

	p.domainEvents = append(p.domainEvents, ProductUpdatedEvent{
		ProductID:    p.ID,
		ProductPubID: p.PublicID,
		Name:         p.Name,
		Description:  p.Description,
		Active:       p.Active,
		UpdatedAt:    now,
	})
}

func (p *Product) GetAndClearEvents() DomainEvents {
	events := p.domainEvents
	p.domainEvents = nil
	return events
}
