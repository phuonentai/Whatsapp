package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
)

const MessageEchoEventType = "instagram.message.echo"

// MessageEcho mirrors a message sent from Meta Business Suite or the
// organization's own Instagram account (is_echo = true). The mirror is
// persisted as an outbound CRM message so the inbox shows what the business
// sent outside the platform.
type MessageEcho struct {
	eventbus.BaseEvent
	OrganizationID int32           `json:"organization_id"`
	MessageSID     string          `json:"message_sid"`
	FromIGUserID   string          `json:"from_ig_user_id"`
	ToIGUserID     string          `json:"to_ig_user_id"`
	MessageType    string          `json:"message_type"`
	Content        string          `json:"content,omitempty"`
	MediaURL       string          `json:"media_url,omitempty"`
	IGTimestamp    time.Time       `json:"ig_timestamp"`
	RawPayload     json.RawMessage `json:"raw_payload"`
}

func NewMessageEcho(
	orgID int32,
	messageSID, fromIGUserID, toIGUserID, messageType, content, mediaURL string,
	igTimestamp time.Time,
	rawPayload json.RawMessage,
) *MessageEcho {
	return &MessageEcho{
		BaseEvent: eventbus.BaseEvent{
			ID:        uuid.New().String(),
			Name:      MessageEchoEventType,
			CreatedAt: time.Now(),
			Meta:      make(map[string]interface{}),
		},
		OrganizationID: orgID,
		MessageSID:     messageSID,
		FromIGUserID:   fromIGUserID,
		ToIGUserID:     toIGUserID,
		MessageType:    messageType,
		Content:        content,
		MediaURL:       mediaURL,
		IGTimestamp:    igTimestamp,
		RawPayload:     rawPayload,
	}
}
