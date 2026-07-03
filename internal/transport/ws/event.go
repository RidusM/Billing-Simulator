package ws

import "time"

type EventType string

const (
	EventSubscriptionCreated  EventType = "subscription.created"
	EventSubscriptionRenewed  EventType = "subscription.renewed"
	EventSubscriptionCanceled EventType = "subscription.canceled"
	EventInvoiceCreated       EventType = "invoice.created"
	EventTimeAdvanced         EventType = "time.advanced"
)

type LiveEvent struct {
	Type      EventType `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Payload   any       `json:"payload"`
}
