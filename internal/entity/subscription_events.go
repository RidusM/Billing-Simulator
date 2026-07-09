package entity

import (
	"time"

	"github.com/google/uuid"
)

type SubscriptionCreatedEvent struct {
	SubscriptionID    uuid.UUID          
	SubscriptionPubID string             
	CustomerID        uuid.UUID          
	CustomerPubID     string             
	PriceID           uuid.UUID          
	PricePubID        string             
	Status            SubscriptionStatus 
	CurrentPeriodEnd  time.Time          
	NextBillingAt     time.Time          
	CreatedAt         time.Time         
}

func (e SubscriptionCreatedEvent) EventType() string      { return "subscription.created" }
func (e SubscriptionCreatedEvent) OccurredOn() time.Time  { return e.CreatedAt }
func (e SubscriptionCreatedEvent) AggregateID() uuid.UUID { return e.SubscriptionID }
func (e SubscriptionCreatedEvent) AggregateType() string  { return "subscription" }

// SubscriptionCanceledEvent
type SubscriptionCanceledEvent struct {
	SubscriptionID    uuid.UUID          
	SubscriptionPubID string             
	CustomerID        uuid.UUID          
	CustomerPubID     string             
	Status            SubscriptionStatus 
	CanceledAt        time.Time          
	AtPeriodEnd       bool               
}

func (e SubscriptionCanceledEvent) EventType() string      { return "subscription.canceled" }
func (e SubscriptionCanceledEvent) OccurredOn() time.Time  { return e.CanceledAt }
func (e SubscriptionCanceledEvent) AggregateID() uuid.UUID { return e.SubscriptionID }
func (e SubscriptionCanceledEvent) AggregateType() string  { return "subscription" }

// SubscriptionRenewedEvent
type SubscriptionRenewedEvent struct {
	SubscriptionID    uuid.UUID     
	SubscriptionPubID string        
	CustomerID        uuid.UUID     
	CustomerPubID     string        
	InvoiceID         uuid.UUID     
	InvoicePubID      string        
	InvoiceAmount     int64        
	InvoiceCurrency   string       
	InvoiceStatus     InvoiceStatus 
	NewPeriodEnd      time.Time    
	RenewedAt         time.Time   
}

func (e SubscriptionRenewedEvent) EventType() string      { return "subscription.renewed" }
func (e SubscriptionRenewedEvent) OccurredOn() time.Time  { return e.RenewedAt }
func (e SubscriptionRenewedEvent) AggregateID() uuid.UUID { return e.SubscriptionID }
func (e SubscriptionRenewedEvent) AggregateType() string  { return "subscription" }
