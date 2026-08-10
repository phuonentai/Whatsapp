package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
)

const MessageSendEventType = "instagram.message.send"

// MessageSend requests asynchronous outbound delivery of a text message
// through the Instagram Graph API. The send is performed by a subscriber
// (CRM outbound sender) with retry and backoff driven by the outbox.
type MessageSend struct {
	eventbus.BaseEvent
	OrganizationID int32           `json:"organization_id"`
	ConversationID int32           `json:"conversation_id"`
	MessageID      int32           `json:"message_id"`
	ToIGUserID     string          `json:"to_ig_user_id"`
	Content        string          `json:"content"`
	RawPayload     json.RawMessage `json:"raw_payload,omitempty"`
}

func NewMessageSend(
	orgID, conversationID, messageID int32,
	toIGUserID, content string,
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
		ToIGUserID:     toIGUserID,
		Content:        content,
	}
}
