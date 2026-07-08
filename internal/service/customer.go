package service

import (
	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"
	"context"
	"fmt"
)

type CustomerRepository interface {
	Create(ctx context.Context, c *entity.Customer) error
	GetByPublicID(ctx context.Context, publicID string) (*entity.Customer, error)
}

type CustomerService struct {
	customers  CustomerRepository
	dispatcher *EventDispatcher
	tm         TransactionManager
	log        logger.Logger
	clock      VirtualClock
}

func NewCustomerService(
	customers CustomerRepository,
	dispatcher *EventDispatcher,
	tm TransactionManager,
	log logger.Logger,
	clock VirtualClock,
) *CustomerService {
	return &CustomerService{
		customers:  customers,
		dispatcher: dispatcher,
		tm:         tm,
		log:        log,
		clock:      clock,
	}
}

func (s *CustomerService) CreateCustomer(ctx context.Context, email string) (*entity.Customer, error) {
	const op = "service.customer.CreateCustomer"

	var customer *entity.Customer

	err := s.tm.ExecuteInTransaction(ctx, op, func(ctx context.Context) error {
		now := s.clock.Now()
		customer = entity.NewCustomer(email, "", now)

		if err := s.customers.Create(ctx, customer); err != nil {
			return fmt.Errorf("create customer: %w", err)
		}

		// Сохраняем события в outbox
		events := customer.GetAndClearEvents()
		if events.HasEvents() {
			if err := s.dispatcher.Dispatch(ctx, events); err != nil {
				return fmt.Errorf("dispatch events: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return customer, nil
}

func (s *CustomerService) GetCustomer(ctx context.Context, publicID string) (*entity.Customer, error) {
	return s.customers.GetByPublicID(ctx, publicID)
}
