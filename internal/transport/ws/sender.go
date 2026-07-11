package webhooksender

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	defaultTimeout   = 10 * time.Second
	maxResponseBytes = 64 * 1024 // не читаем безлимитно ответ чужого сервера
)

// HTTPSender реализует service.WebhookSender поверх net/http.
// Специально НЕ переиспользует http.DefaultClient — у симулятора должны быть свои
// таймауты, иначе один зависший клиентский эндпоинт держит воркер (и всю очередь ретраев) вечно.
type HTTPSender struct {
	client *http.Client
}

func NewHTTPSender() *HTTPSender {
	return &HTTPSender{
		client: &http.Client{
			Timeout: defaultTimeout,
			// Явно не следуем редиректам: вебхук должен идти на URL, который явно указал клиент,
			// а не туда, куда его перенаправит скомпрометированный/неправильно настроенный сервер.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Send реализует service.WebhookSender: отправляет подписанный payload на url клиента,
// возвращает HTTP-статус ответа (используется вызывающим кодом для решения retry/success).
func (s *HTTPSender) Send(ctx context.Context, url string, payload []byte, signature string, timestamp int64) (int, error) {
	const op = "webhooksender.HTTPSender.Send"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return 0, fmt.Errorf("%s: build request: %w", op, err)
	}

	req.Header.Set("Content-Type", "application/json")
	// Заголовки в стиле Stripe — signature содержит "v1,<hmac>", timestamp — unix seconds,
	// подпись как раз строится как fmt.Sprintf("%d.%s", timestamp, payload) на стороне entity.WebhookEndpoint.
	req.Header.Set("X-Billing-Signature", signature)
	req.Header.Set("X-Billing-Timestamp", strconv.FormatInt(timestamp, 10))

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	defer resp.Body.Close()

	// Тело ответа реально не нужно вызывающему коду, но его обязательно надо вычитать
	// (и ограничить размер), иначе соединение не уйдёт обратно в пул на переиспользование.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBytes))

	return resp.StatusCode, nil
}
