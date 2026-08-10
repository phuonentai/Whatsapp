package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
)

const MessageSendEventType = "whatsapp.message.send"

// MessageSend requests asynchronous outbound delivery of a text message
// through the WhatsApp Cloud API. The send is performed by a subscriber
// (CRM outbound sender) with retry and backoff driven by the outbox.
type MessageSend struct {
	eventbus.BaseEvent
	OrganizationID int32           `json:"organization_id"`
	ConversationID int32           `json:"conversation_id"`
	MessageID      int32           `json:"message_id"`
	To             string          `json:"to"`
	Content        string          `json:"content"`
	RawPayload     json.RawMessage `json:"raw_payload,omitempty"`
}

// NewMessageSend builds a durable outbound-send request.
func NewMessageSend(
	orgID, conversationID, messageID int32,
	to, content string,
) *MessageSend {
	return &MessageSend{
		BaseEvent: eventbus.BaseEvent{
			ID:        uuid.New().String(),
			Name:      MessageSendEventType,
			CreatedAt: time.Now(),
			Meta:      make(map[string]interface{}),
		},
		OrganizationID: orgID,
		ConversationID: conversationID,
		MessageID:      messageID,
		To:             to,
		Content:        content,
	}
}
