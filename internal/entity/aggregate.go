package entity

type AggregateRoot struct {
	domainEvents DomainEvents
}

func (a *AggregateRoot) Raise(event DomainEvent) {
	a.domainEvents.Raise(event)
}

func (a *AggregateRoot) Events() DomainEvents {
	return a.domainEvents.ClearAndReturn()
}
