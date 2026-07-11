package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"bill-stripe-sim/internal/entity"

	"github.com/google/uuid"
)

// WebhookEndpointCreator — интерфейс объявлен здесь, у потребителя (WebhookEndpointService).
type WebhookEndpointCreator interface {
	Create(ctx context.Context, e *entity.WebhookEndpoint) error
	GetByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*entity.WebhookEndpoint, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

type WebhookEndpointService struct {
	endpoints WebhookEndpointCreator
	tm        TransactionManager
	clock     VirtualClock
}

func NewWebhookEndpointService(endpoints WebhookEndpointCreator, tm TransactionManager, clock VirtualClock) *WebhookEndpointService {
	return &WebhookEndpointService{endpoints: endpoints, tm: tm, clock: clock}
}

// CreateEndpoint регистрирует новый webhook-эндпоинт клиента. Секрет генерируется один раз
// и возвращается вызывающему коду ТОЛЬКО в этот момент (entity.WebhookEndpoint.Secret) —
// репозиторий обязан сохранить его исключительно как secret_encrypted, никогда plaintext.
func (s *WebhookEndpointService) CreateEndpoint(
	ctx context.Context,
	customerID uuid.UUID,
	url string,
	description string,
	enabledEvents []string,
) (*entity.WebhookEndpoint, error) {
	const op = "service.WebhookEndpointService.CreateEndpoint"

	secret, prefix, err := generateWebhookSecret()
	if err != nil {
		return nil, fmt.Errorf("%s: generate secret: %w", op, err)
	}

	var ep *entity.WebhookEndpoint
	err = s.tm.ExecuteInTransaction(ctx, op, func(ctx context.Context) error {
		ep = entity.NewWebhookEndpoint(customerID, url, prefix, secret, s.clock.Now())
		if len(enabledEvents) > 0 {
			ep.EnabledEvents = enabledEvents
		}
		ep.Description = description
		return s.endpoints.Create(ctx, ep)
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return ep, nil
}

func (s *WebhookEndpointService) ListForCustomer(ctx context.Context, customerID uuid.UUID) ([]*entity.WebhookEndpoint, error) {
	return s.endpoints.GetByCustomerID(ctx, customerID)
}

func (s *WebhookEndpointService) DeleteEndpoint(ctx context.Context, id uuid.UUID) error {
	return s.endpoints.SoftDelete(ctx, id)
}

// generateWebhookSecret — секрет в стиле Stripe: "whsec_" + 32 случайных байта в hex,
// plus короткий prefix (первые 8 символов) для отображения в UI без раскрытия полного секрета.
func generateWebhookSecret() (secret, prefix string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil { // Убрана лишняя запятая
		return "", "", fmt.Errorf("generate secret: %w", err)
	}
	secret = "whsec_" + hex.EncodeToString(b) // Добавлено подчеркивание, как в Stripe
	prefix = secret[:14]                      // "whsec_" (6) + 8 hex-символов = 14
	return secret, prefix, nil
}
