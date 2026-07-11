package service

import (
	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"
	"context"
	"fmt"

	"github.com/google/uuid"
)

// PriceRepository — интерфейс объявлен ЗДЕСЬ, в service пакете
type PriceRepository interface {
	Create(ctx context.Context, p *entity.Price) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Price, error)
	GetByPublicID(ctx context.Context, publicID string) (*entity.Price, error)
	List(ctx context.Context, limit, offset int) ([]*entity.Price, error)
	Update(ctx context.Context, p *entity.Price) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

type PriceService struct {
	prices PriceRepository // ← Локальный интерфейс!
	tm     TransactionManager
	log    logger.Logger
	clock  VirtualClock
}

func NewPriceService(
	prices PriceRepository, // ← Локальный интерфейс!
	tm TransactionManager,
	log logger.Logger,
	clock VirtualClock,
) *PriceService {
	return &PriceService{
		prices: prices,
		tm:     tm,
		log:    log,
		clock:  clock,
	}
}

func (s *PriceService) CreatePrice(
	ctx context.Context,
	productID uuid.UUID,
	amount int64,
	currency string,
	interval entity.BillingInterval,
	intervalCount int,
) (*entity.Price, error) {
	const op = "service.price.CreatePrice"

	var price *entity.Price

	err := s.tm.ExecuteInTransaction(ctx, op, func(ctx context.Context) error {
		var err error
		price, err = entity.NewPrice(productID, amount, currency, interval, intervalCount, s.clock.Now())
		if err != nil {
			return fmt.Errorf("create price entity: %w", err)
		}
		return s.prices.Create(ctx, price)
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return price, nil
}

func (s *PriceService) GetPrice(ctx context.Context, id uuid.UUID) (*entity.Price, error) {
	return s.prices.GetByID(ctx, id)
}

func (s *PriceService) ListPrices(ctx context.Context, limit, offset int) ([]*entity.Price, error) {
	return s.prices.List(ctx, limit, offset)
}

func (s *PriceService) DeletePrice(ctx context.Context, id uuid.UUID) error {
	return s.prices.SoftDelete(ctx, id)
}
