package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"

	"github.com/forKate/ws-server/internal/kafka"
	"github.com/forKate/ws-server/internal/message"
)

const (
	writeWait      = 5 * time.Second
	sendBufferSize = 16
)

type Client struct {
	UserID   string
	conn     *websocket.Conn
	hub      *Hub
	producer kafka.Producer
	logger   *slog.Logger
	Send     chan []byte

	closeOnce sync.Once
}

func NewClient(userID string, conn *websocket.Conn, hub *Hub, producer kafka.Producer, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}

	client := &Client{
		UserID:   userID,
		conn:     conn,
		hub:      hub,
		producer: producer,
		logger:   logger,
		Send:     make(chan []byte, sendBufferSize),
	}

	hub.Register(client)
	return client
}

func (c *Client) Run(ctx context.Context) {
	defer func() {
		c.hub.Unregister(c)
		c.CloseWithError(nil)
	}()

	go c.writePump(ctx)
	c.readPump(ctx)
}

func (c *Client) readPump(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			c.logger.Error("readPump panic", "panic", r)
		}
	}()

	for {
		if ctx.Err() != nil {
			return
		}

		_, payload, err := c.conn.ReadMessage()
		if err != nil {
			c.logger.Warn("read message failed", "err", err)
			return
		}

		var incoming message.InboundMessage
		if err := json.Unmarshal(payload, &incoming); err != nil {
			c.logger.Warn("invalid payload", "err", err)
			continue
		}

		if len(incoming.Recipients) == 0 {
			c.logger.Warn("missing recipients", "user_id", c.UserID)
			continue
		}

		envelope := message.NewEnvelope(c.UserID, incoming.Recipients, incoming.Payload)
		if err := c.producer.Publish(ctx, envelope); err != nil {
			c.logger.Error("publish to kafka failed", "err", err)
		}
	}
}

func (c *Client) writePump(ctx context.Context) {
	defer c.conn.Close()

	for {
		select {
		case msg, ok := <-c.Send:
			if !ok {
				c.writeClose(websocket.CloseNormalClosure, "shutdown")
				return
			}

			if err := c.write(websocket.TextMessage, msg); err != nil {
				if !websocket.IsUnexpectedCloseError(err) {
					c.logger.Error("write failed", "err", err)
				}
				return
			}
		case <-ctx.Done():
			c.writeClose(websocket.CloseNormalClosure, "context cancelled")
			return
		}
	}
}

func (c *Client) write(messageType int, data []byte) error {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteMessage(messageType, data)
}

func (c *Client) CloseWithError(err error) {
	c.closeOnce.Do(func() {
		close(c.Send)
		if err != nil {
			c.logger.Warn("client closed", "err", err)
		}
	})
}

func (c *Client) writeClose(code int, reason string) {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	_ = c.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(writeWait))
}
