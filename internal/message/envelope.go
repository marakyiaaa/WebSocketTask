package message

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// InboundMessage входящее сообщение от клиента
type InboundMessage struct {
	Recipients []string        `json:"recipients"`
	Payload    json.RawMessage `json:"payload"`
}

// Envelope сообщение, которое доставляется клиентам.
type Envelope struct {
	ID         string          `json:"id"`
	SenderID   string          `json:"sender_id"`
	Recipients []string        `json:"recipients"`
	Payload    json.RawMessage `json:"payload"`
	SentAt     time.Time       `json:"sent_at"`
}

func NewEnvelope(sender string, recipients []string, payload json.RawMessage) Envelope {
	return Envelope{
		ID:         uuid.NewString(),
		SenderID:   sender,
		Recipients: append([]string(nil), recipients...),
		Payload:    payload,
		SentAt:     time.Now().UTC(),
	}
}
