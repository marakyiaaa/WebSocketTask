package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/IBM/sarama"

	"github.com/marakyiaaa/WebSocketTask/internal/message"
)

type Producer interface {
	Publish(ctx context.Context, envelope message.Envelope) error
	Close() error
}

type syncProducer struct {
	topic    string
	producer sarama.SyncProducer
	logger   *slog.Logger
}

func NewProducer(brokers []string, topic string, logger *slog.Logger) (Producer, error) {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_8_1_0
	cfg.Producer.Return.Successes = true
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Retry.Max = 5

	prod, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return nil, fmt.Errorf("create sync producer: %w", err)
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &syncProducer{topic: topic, producer: prod, logger: logger}, nil
}

func (p *syncProducer) Publish(ctx context.Context, envelope message.Envelope) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	payload, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshaling: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(envelope.SenderID),
		Value: sarama.ByteEncoder(payload),
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("send kafka message: %w", err)
	}

	p.logger.Debug("message published", "partition", partition, "offset", offset)
	return nil
}

func (p *syncProducer) Close() error {
	return p.producer.Close()
}
