package ws

import (
	"context"
	"log/slog"
	"strings"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"

	"github.com/forKate/ws-server/internal/auth"
	"github.com/forKate/ws-server/internal/kafka"
)

const (
	ctxUserKey     = "ws:user"
	defaultBufSize = 1024 * 1024
)

type Handler struct {
	baseCtx   context.Context
	hub       *Hub
	validator *auth.Validator
	producer  kafka.Producer
	logger    *slog.Logger
}

func NewHandler(baseCtx context.Context, hub *Hub, validator *auth.Validator, producer kafka.Producer, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	return &Handler{
		baseCtx:   baseCtx,
		hub:       hub,
		validator: validator,
		producer:  producer,
		logger:    logger,
	}
}
func (h *Handler) Register(app *fiber.App) {
	app.Use("/ws", h.authorize)
	app.Get("/ws", websocket.New(h.handleConn, websocket.Config{
		ReadBufferSize:  defaultBufSize,
		WriteBufferSize: defaultBufSize,
	}))
}

func (h *Handler) authorize(c *fiber.Ctx) error {
	if !websocket.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}

	token, err := h.extractToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}

	claims, err := h.validator.Validate(c.Context(), token)
	if err != nil {
		h.logger.Warn("token validation failed", "err", err)
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	c.Locals(ctxUserKey, claims.UserID)
	return c.Next()
}

func (h *Handler) handleConn(conn *websocket.Conn) {
	ctx, cancel := context.WithCancel(h.baseCtx)
	defer cancel()

	userID, _ := conn.Locals(ctxUserKey).(string)
	if strings.TrimSpace(userID) == "" {
		h.logger.Error("websocket user missing, closing connection")
		_ = conn.Close()
		return
	}

	client := NewClient(userID, conn, h.hub, h.producer, h.logger)
	client.Run(ctx)
}

func (h *Handler) extractToken(c *fiber.Ctx) (string, error) {
	if header := c.Get(fiber.HeaderAuthorization); header != "" {
		return auth.ExtractBearerToken(header)
	}

	if token := c.Query("token"); token != "" {
		return token, nil
	}

	if token := c.Cookies("token"); token != "" {
		return token, nil
	}

	return "", auth.ErrMissingToken
}
