package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"

	"github.com/marakyiaaa/WebSocketTask/internal/message"
)

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]struct{}
	logger  *slog.Logger
}

func NewHub(logger *slog.Logger) *Hub {
	if logger == nil {
		logger = slog.Default()
	}

	return &Hub{
		clients: make(map[string]map[*Client]struct{}),
		logger:  logger,
	}
}
func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.UserID]; !ok {
		h.clients[client.UserID] = make(map[*Client]struct{})
	}
	h.clients[client.UserID][client] = struct{}{}
	h.logger.Info("client connected", "user_id", client.UserID, "active", len(h.clients[client.UserID]))
}
func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	peers, ok := h.clients[client.UserID]
	if !ok {
		return
	}

	delete(peers, client)
	if len(peers) == 0 {
		delete(h.clients, client.UserID)
	}
	h.logger.Info("client disconnected", "user_id", client.UserID)
}

func (h *Hub) Deliver(ctx context.Context, envelope message.Envelope) error {
	payload, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	delivered := 0
	for _, recipient := range envelope.Recipients {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		for _, client := range h.snapshot(recipient) {
			select {
			case client.Send <- payload:
				delivered++
			default:
				go client.CloseWithError(errors.New("send buffer overflow"))
			}
		}
	}

	if delivered == 0 {
		return errors.New("recipient offline")
	}

	return nil
}

func (h *Hub) snapshot(userID string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	peers := h.clients[userID]
	if len(peers) == 0 {
		return nil
	}

	list := make([]*Client, 0, len(peers))
	for client := range peers {
		list = append(list, client)
	}
	return list
}
