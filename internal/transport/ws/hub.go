package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"bill-stripe-sim/pkg/logger"
)

// Envelope — единый конверт для всех сообщений, летящих в dashboard.
// Type используется фронтендом для роутинга (api_request / webhook_log / domain_event / clock).
type Envelope struct {
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
}

// Hub — держит набор активных клиентов и рассылает им сообщения.
// Один Hub на процесс, потокобезопасен.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}

	register   chan *Client
	unregister chan *Client
	broadcast  chan Envelope

	log logger.Logger
}

func NewHub(log logger.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Envelope, 256),
		log:        log,
	}
}

// Run — блокирующий цикл обработки Hub'а. Запускать в отдельной горутине при старте приложения.
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = struct{}{}
			h.mu.Unlock()
			h.log.Info("ws client connected", "total", h.clientCount())

		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()
			h.log.Info("ws client disconnected", "total", h.clientCount())

		case env := <-h.broadcast:
			data, err := json.Marshal(env)
			if err != nil {
				h.log.Error("ws: failed to marshal envelope", "error", err, "type", env.Type)
				continue
			}
			h.mu.RLock()
			for c := range h.clients {
				select {
				case c.send <- data:
				default:
					// Клиент не успевает читать — не блокируем Hub, дропаем соединение.
					// Само закрытие произойдёт через writePump -> unregister.
					h.log.Warn("ws client send buffer full, dropping", "type", env.Type)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast — неблокирующая отправка события всем подключённым клиентам.
// Безопасно вызывать из любой горутины (HTTP middleware, OutboxProcessor, WebhookDeliveryService).
func (h *Hub) Broadcast(eventType string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		h.log.Error("ws: failed to marshal payload", "error", err, "type", eventType)
		return
	}
	env := Envelope{Type: eventType, Payload: data, Timestamp: time.Now().UTC()}

	select {
	case h.broadcast <- env:
	default:
		h.log.Warn("ws hub broadcast channel full, dropping event", "type", eventType)
	}
}

func (h *Hub) clientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
