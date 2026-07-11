package service

import (
	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type PaymentIntentRepository interface {
	Create(ctx context.Context, pi *entity.PaymentIntent) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.PaymentIntent, error)
	GetByPublicID(ctx context.Context, publicID string) (*entity.PaymentIntent, error)
	GetByInvoiceID(ctx context.Context, invoiceID uuid.UUID) (*entity.PaymentIntent, error)
	Update(ctx context.Context, pi *entity.PaymentIntent) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

type PaymentService struct {
	paymentIntents PaymentIntentRepository
	invoices       InvoiceRepository
	dispatcher     *EventDispatcher
	tm             TransactionManager
	log            logger.Logger
	clock          VirtualClock
	rateManager    *PaymentRateManager
}

func NewPaymentService(
	paymentIntents PaymentIntentRepository,
	invoices InvoiceRepository,
	dispatcher *EventDispatcher,
	tm TransactionManager,
	log logger.Logger,
	clock VirtualClock,
	rateManager *PaymentRateManager,
) *PaymentService {
	return &PaymentService{
		paymentIntents: paymentIntents,
		invoices:       invoices,
		dispatcher:     dispatcher,
		tm:             tm,
		log:            log,
		clock:          clock,
		rateManager:    rateManager,
	}
}

func (s *PaymentService) CreatePaymentIntent(ctx context.Context, invoiceID uuid.UUID) (*entity.PaymentIntent, error) {
	const op = "service.payment.CreatePaymentIntent"

	var pi *entity.PaymentIntent
	err := s.tm.ExecuteInTransaction(ctx, op, func(ctx context.Context) error {
		inv, err := s.invoices.GetByID(ctx, invoiceID)
		if err != nil {
			return fmt.Errorf("get invoice: %w", err)
		}

		var err2 error
		pi, err2 = entity.NewPaymentIntent(inv.CustomerID, &inv.ID, inv.Amount, inv.Currency, s.clock.Now())
		if err2 != nil {
			return fmt.Errorf("create payment intent entity: %w", err2)
		}

		return s.paymentIntents.Create(ctx, pi)
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return pi, nil
}

func (s *PaymentService) ConfirmPayment(ctx context.Context, publicID string) (*entity.PaymentIntent, error) {
	const op = "service.payment.ConfirmPayment"
	var pi *entity.PaymentIntent
	var inv *entity.Invoice
	err := s.tm.ExecuteInTransaction(ctx, op, func(ctx context.Context) error {
		var err error
		pi, err = s.paymentIntents.GetByPublicID(ctx, publicID)
		if err != nil {
			return fmt.Errorf("get payment intent: %w", err)
		}

		if simulatePayment(ctx, s.rateManager) == entity.InvoiceStatusPaid {
			pi.MarkSucceeded(s.clock.Now())

			// Обновить инвойс
			inv, err = s.invoices.GetByID(ctx, *pi.InvoiceID)
			if err != nil {
				return fmt.Errorf("get invoice: %w", err)
			}
			if err := inv.MarkPaid(s.clock.Now()); err != nil {
				return fmt.Errorf("mark invoice paid: %w", err)
			}
			if err := s.invoices.Update(ctx, inv); err != nil {
				return fmt.Errorf("update invoice: %w", err)
			}
		} else {
			pi.MarkFailed(s.clock.Now(), "card_declined", "insufficient_funds")

			// Также нужно обновить инвойс как failed
			inv, err = s.invoices.GetByID(ctx, *pi.InvoiceID)
			if err != nil {
				return fmt.Errorf("get invoice: %w", err)
			}

			isFinalAttempt := (inv.AttemptCount + 1) >= 3
			if err := inv.MarkPaymentFailed(s.clock.Now(), "card_declined", isFinalAttempt); err != nil {
				return fmt.Errorf("mark payment failed: %w", err)
			}
			if err := s.invoices.Update(ctx, inv); err != nil {
				return fmt.Errorf("update invoice: %w", err)
			}
		}
		if err := s.paymentIntents.Update(ctx, pi); err != nil {
			return fmt.Errorf("update payment intent: %w", err)
		}
		// ОТПРАВЛЯЕМ СОБЫТИЯ
		piEvents := pi.GetAndClearEvents()
		invEvents := inv.GetAndClearEvents()
		allEvents := append(piEvents, invEvents...)
		if allEvents.HasEvents() {
			if err := s.dispatcher.Dispatch(ctx, allEvents); err != nil {
				return fmt.Errorf("dispatch events: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return pi, nil
}
