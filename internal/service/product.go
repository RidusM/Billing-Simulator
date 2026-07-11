package service

import (
	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/logger"
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ProductRepository — интерфейс объявлен ЗДЕСЬ, в service пакете
type ProductRepository interface {
	Create(ctx context.Context, p *entity.Product) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Product, error)
	GetByPublicID(ctx context.Context, publicID string) (*entity.Product, error)
	List(ctx context.Context, limit, offset int) ([]*entity.Product, error)
	Update(ctx context.Context, p *entity.Product) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

type ProductService struct {
	products ProductRepository // ← Локальный интерфейс!
	tm       TransactionManager
	log      logger.Logger
	clock    VirtualClock
}

func NewProductService(
	products ProductRepository, // ← Локальный интерфейс!
	tm TransactionManager,
	log logger.Logger,
	clock VirtualClock,
) *ProductService {
	return &ProductService{
		products: products,
		tm:       tm,
		log:      log,
		clock:    clock,
	}
}

func (s *ProductService) CreateProduct(ctx context.Context, name, description string) (*entity.Product, error) {
	const op = "service.product.CreateProduct"

	var product *entity.Product

	err := s.tm.ExecuteInTransaction(ctx, op, func(ctx context.Context) error {
		var err error
		product, err = entity.NewProduct(name, description, s.clock.Now())
		if err != nil {
			return fmt.Errorf("create product entity: %w", err)
		}
		return s.products.Create(ctx, product)
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return product, nil
}

func (s *ProductService) GetProduct(ctx context.Context, publicID string) (*entity.Product, error) {
	return s.products.GetByPublicID(ctx, publicID)
}

func (s *ProductService) ListProducts(ctx context.Context, limit, offset int) ([]*entity.Product, error) {
	return s.products.List(ctx, limit, offset)
}

func (s *ProductService) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	return s.products.SoftDelete(ctx, id)
}
