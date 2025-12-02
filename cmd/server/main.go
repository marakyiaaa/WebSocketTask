package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"

	"github.com/forKate/ws-server/internal/auth"
	"github.com/forKate/ws-server/internal/config"
	"github.com/forKate/ws-server/internal/kafka"
	"github.com/forKate/ws-server/internal/ws"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg := config.Load()
	logger.Info("starting websocket server", "http", cfg.HTTPAddr, "topic", cfg.Kafka.Topic, "group", cfg.Kafka.Group)

	validator := auth.NewValidator(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience, cfg.JWTLeeway)
	hub := ws.NewHub(logger)

	producer, err := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic, logger)
	if err != nil {
		logger.Error("unable to create producer", "err", err)
		os.Exit(1)
	}
	defer producer.Close()

	consumer, err := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topic, cfg.Kafka.Group, hub, logger)
	if err != nil {
		logger.Error("unable to create consumer", "err", err)
		os.Exit(1)
	}
	defer consumer.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	consumer.Start(ctx)

	app := fiber.New()
	wsHandler := ws.NewHandler(ctx, hub, validator, producer, logger)
	wsHandler.Register(app)

	go func() {
		<-ctx.Done()
		logger.Info("shutting down")
		if err := app.Shutdown(); err != nil {
			logger.Error("shutdown error", "err", err)
		}
	}()

	if err := app.Listen(cfg.HTTPAddr); err != nil {
		if ctx.Err() == nil {
			logger.Error("listen error", "err", err)
			os.Exit(1)
		}
	}

	logger.Info("server stopped")
}
