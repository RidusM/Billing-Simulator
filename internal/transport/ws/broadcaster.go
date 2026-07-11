package wstransport

import (
	"bill-stripe-sim/pkg/websocket"
)

// HubBroadcaster реализует все Broadcaster-интерфейсы, которые нужно завести в service-пакете
// (см. патчи ниже) — просто прокидывает вызовы в ws.Hub.Broadcast, которая уже потокобезопасна
// и неблокирующая.
type HubBroadcaster struct {
	hub *websocket.Hub
}

func NewHubBroadcaster(hub *websocket.Hub) *HubBroadcaster {
	return &HubBroadcaster{hub: hub}
}

func (b *HubBroadcaster) Broadcast(eventType string, payload any) {
	b.hub.Broadcast(eventType, payload)
}

/*
ТРИ МЕСТА, КУДА НУЖНО ВРУЧНУЮ ДОБАВИТЬ ВЫЗОВЫ (интерфейс объявляется у потребителя,
поэтому это разные типы, хоть реализация одна — HubBroadcaster сверху):

1) service/notification.go — активность по доменным событиям в общую ленту дашборда:

	type Broadcaster interface {
	    Broadcast(eventType string, payload any)
	}

	type NotificationService struct {
	    sender      EventSender
	    webhook     WebhookDispatcher
	    broadcaster Broadcaster // ← добавить
	    log         logger.Logger
	}

	func NewNotificationService(sender EventSender, webhook WebhookDispatcher, broadcaster Broadcaster, log logger.Logger) *NotificationService {
	    return &NotificationService{sender: sender, webhook: webhook, broadcaster: broadcaster, log: log}
	}

	func (s *NotificationService) handleEvent(ctx context.Context, customerID uuid.UUID, eventType entity.EventType, payload []byte) error {
	    s.broadcaster.Broadcast("domain_event", map[string]any{
	        "event_type":  string(eventType),
	        "customer_id": customerID,
	        "payload":     json.RawMessage(payload),
	    })
	    // ... остальное как было (Kafka + webhook)
	}

2) service/webhook.go — полная цепочка "pending -> delivered/failed" для дашборда:

	type Broadcaster interface {
	    Broadcast(eventType string, payload any)
	}

	// добавить поле broadcaster в WebhookDeliveryService и параметр в конструктор,
	// затем в deliverToEndpoint после каждого s.logs.Create / s.logs.Update:

	    s.broadcaster.Broadcast("webhook_log", logEntry)

3) transport/http middleware (логирование запроса, см. http/middleware.go loggingMiddleware) —
   в конце функции, после подсчёта duration/status, добавить:

	    h.broadcaster.Broadcast("api_request", map[string]any{
	        "method":   c.Request.Method,
	        "path":     c.FullPath(),
	        "status":   status,
	        "duration": duration.Milliseconds(),
	    })

   Это и есть третье звено цепочки "запрос разработчика -> ответ симулятора -> вебхук",
   которую вы обещаете в питче — без него дашборд покажет только 2 из 3 событий.
*/
