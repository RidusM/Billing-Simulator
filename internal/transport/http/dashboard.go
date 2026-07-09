package handler

import (
	"net/http"

	"bill-stripe-sim/pkg/logger"
	ws "bill-stripe-sim/pkg/websocket"

	"github.com/gorilla/websocket"
)

// DashboardHandler отдаёт /ws — апгрейд до WebSocket для live-дашборда.
type DashboardHandler struct {
	hub      *ws.Hub
	upgrader websocket.Upgrader
	log      logger.Logger
}

func NewDashboardHandler(hub *ws.Hub, log logger.Logger) *DashboardHandler {
	return &DashboardHandler{
		hub: hub,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// Локальный симулятор для разработчиков: дашборд открывается с того же origin
			// (docker-compose), поэтому CORS на WS-уровне не нужен. Если фронт хостится отдельно —
			// сузить список origin явно вместо "разрешить всё".
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		log: log,
	}
}

func (h *DashboardHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.LogAttrs(r.Context(), logger.WarnLevel, "ws upgrade failed", logger.Any("error", err))
		return
	}

	client := ws.NewClient(h.hub, conn)
	client.Serve() // блокируется до дисконнекта клиента
}
