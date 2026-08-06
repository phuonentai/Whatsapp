package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
)

const MessageReceivedEventType = "whatsapp.message.received"

type MessageReceived struct {
	eventbus.BaseEvent
	OrganizationID  int32               `json:"organization_id"`
	ContactID       int32               `json:"contact_id,omitempty"`
	ConversationID  int32               `json:"conversation_id,omitempty"`
	MessageID       int32               `json:"message_id,omitempty"`
	MessageSID      string              `json:"message_sid"`
	From            string              `json:"from"`
	To              string              `json:"to"`
	MessageType     string              `json:"message_type"`
	Content         string              `json:"content,omitempty"`
	MediaURL        string              `json:"media_url,omitempty"`
	WhatsAppTimestamp time.Time         `json:"whatsapp_timestamp"`
	RawPayload      json.RawMessage     `json:"raw_payload"`
}

func NewMessageReceived(
	orgID int32,
	messageSID, from, to, messageType, content, mediaURL string,
	whatsappTimestamp time.Time,
	rawPayload json.RawMessage,
) *MessageReceived {
	return &MessageReceived{
		BaseEvent: eventbus.BaseEvent{
			ID:        uuid.New().String(),
			Name:      MessageReceivedEventType,
			CreatedAt: time.Now(),
			Meta:      make(map[string]interface{}),
		},
		OrganizationID:    orgID,
		MessageSID:        messageSID,
		From:              from,
		To:                to,
		MessageType:       messageType,
		Content:           content,
		MediaURL:          mediaURL,
		WhatsAppTimestamp: whatsappTimestamp,
		RawPayload:        rawPayload,
	}
}
