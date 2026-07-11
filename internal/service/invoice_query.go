package service

import (
	"context"

	"bill-stripe-sim/internal/entity"

	"github.com/google/uuid"
)

// InvoiceReader — интерфейс объявлен здесь, у потребителя (InvoiceQueryService).
// Специально отделён от InvoiceRepository (billing.go), у которого только Create/GetByID/Update
// нужные для транзакционных операций — здесь чистое чтение для HTTP GET-эндпоинтов.
type InvoiceReader interface {
	GetByPublicID(ctx context.Context, publicID string) (*entity.Invoice, error)
	GetByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*entity.Invoice, error)
}

type InvoiceQueryService struct {
	invoices InvoiceReader
}

func NewInvoiceQueryService(invoices InvoiceReader) *InvoiceQueryService {
	return &InvoiceQueryService{invoices: invoices}
}

func (s *InvoiceQueryService) GetInvoice(ctx context.Context, publicID string) (*entity.Invoice, error) {
	return s.invoices.GetByPublicID(ctx, publicID)
}

func (s *InvoiceQueryService) ListForCustomer(ctx context.Context, customerID uuid.UUID) ([]*entity.Invoice, error) {
	return s.invoices.GetByCustomerID(ctx, customerID)
}
