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
	AggregateRoot
}

func NewProduct(name, description string, now time.Time) (*Product, error) {
	pubID, err := GeneratePublicID("prod")
	if err != nil {
		return nil, err
	}

	utc := now.UTC()

	p := &Product{
		ID:          uuid.New(),
		PublicID:    pubID,
		Name:        name,
		Description: description,
		Active:      true,
		Metadata:    NewMetadata(),
		CreatedAt:   utc,
		UpdatedAt:   utc,
	}

	p.Raise(ProductCreatedEvent{
		ProductID:    p.ID,
		ProductPubID: p.PublicID,
		Name:         p.Name,
		Description:  p.Description,
		CreatedAt:    utc,
	})

	return p, nil
}

func (p *Product) Update(
	name,
	description string,
	active bool,
	now time.Time,
) {

	if p.Name == name &&
		p.Description == description &&
		p.Active == active {
		return
	}

	utc := now.UTC()

	p.Name = name
	p.Description = description
	p.Active = active
	p.UpdatedAt = utc

	p.Raise(ProductUpdatedEvent{
		ProductID:    p.ID,
		ProductPubID: p.PublicID,
		Name:         p.Name,
		Description:  p.Description,
		Active:       p.Active,
		UpdatedAt:    utc,
	})
}

func (p *Product) GetAndClearEvents() DomainEvents {
	return p.domainEvents.ClearAndReturn()
}
