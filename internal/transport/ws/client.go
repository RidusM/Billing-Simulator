package websocket

import (
	"time"

	"github.com/gorilla/websocket"
)

// Требуется зависимость: github.com/gorilla/websocket
// go get github.com/gorilla/websocket

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096 // dashboard только читает, входящие сообщения нам не нужны — держим лимит маленьким
	sendBufferSize = 64
)

// Client — один подключённый dashboard (браузер). Соединение однонаправленное:
// сервер пушит события, клиент не отправляет данные (кроме pong/close).
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, sendBufferSize),
	}
}

// Serve — регистрирует клиента в Hub и запускает read/write pump.
// Блокируется до закрытия соединения. Вызывать в горутине из HTTP-хендлера.
func (c *Client) Serve() {
	c.hub.register <- c

	go c.writePump()
	c.readPump()
}

// readPump — нужен только для того, чтобы отлавливать закрытие соединения и pong'и.
// Dashboard ничего не присылает, поэтому любое входящее сообщение просто игнорируется.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case data, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub закрыл канал — отправляем close-фрейм и выходим.
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
