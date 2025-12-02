package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/IBM/sarama"

	"github.com/marakyiaaa/WebSocketTask/internal/message"
)

type Dispatcher interface {
	Deliver(ctx context.Context, envelope message.Envelope) error
}

type Consumer struct {
	group      sarama.ConsumerGroup
	topic      string
	dispatcher Dispatcher
	logger     *slog.Logger
	once       sync.Once
}

func NewConsumer(brokers []string, topic, groupID string, dispatcher Dispatcher, logger *slog.Logger) (*Consumer, error) {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_8_1_0
	cfg.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	cfg.Consumer.Offsets.Initial = sarama.OffsetNewest

	grp, err := sarama.NewConsumerGroup(brokers, groupID, cfg)
	if err != nil {
		return nil, fmt.Errorf("create consumer group: %w", err)
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &Consumer{
		group:      grp,
		topic:      topic,
		dispatcher: dispatcher,
		logger:     logger,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) {
	handler := &groupHandler{dispatcher: c.dispatcher, logger: c.logger}

	go func() {
		for {
			if err := c.group.Consume(ctx, []string{c.topic}, handler); err != nil {
				c.logger.Error("consumer loop error", "err", err)
			}

			if ctx.Err() != nil {
				return
			}
		}
	}()
}

func (c *Consumer) Close() error {
	var err error
	c.once.Do(func() {
		err = c.group.Close()
	})
	return err
}

type groupHandler struct {
	dispatcher Dispatcher
	logger     *slog.Logger
}

// Setup Cleanup interface sarama
func (h *groupHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *groupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *groupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var envelope message.Envelope
		if err := json.Unmarshal(msg.Value, &envelope); err != nil {
			h.logger.Error("failed to decode", "err", err)
			session.MarkMessage(msg, "decode-error")
			continue
		}

		if err := h.dispatcher.Deliver(session.Context(), envelope); err != nil {
			h.logger.Error("failed to deliver message", "err", err)
			continue
		}

		session.MarkMessage(msg, "delivered")
	}
	return nil
}
