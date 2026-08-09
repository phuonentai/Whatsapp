package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
)

const MessageEchoEventType = "whatsapp.message.echo"

// MessageEcho mirrors a message the organization's own phone WhatsApp Business
// app sent to a customer (coexistence). The mirror is persisted as an outbound
// CRM message so the inbox shows what the phone app sent.
type MessageEcho struct {
	eventbus.BaseEvent
	OrganizationID    int32           `json:"organization_id"`
	MessageSID        string          `json:"message_sid"`
	From              string          `json:"from"`
	To                string          `json:"to"`
	MessageType       string          `json:"message_type"`
	Content           string          `json:"content,omitempty"`
	MediaURL          string          `json:"media_url,omitempty"`
	WhatsAppTimestamp time.Time       `json:"whatsapp_timestamp"`
	RawPayload        json.RawMessage `json:"raw_payload"`
}

func NewMessageEcho(
	orgID int32,
	messageSID, from, to, messageType, content, mediaURL string,
	whatsappTimestamp time.Time,
	rawPayload json.RawMessage,
) *MessageEcho {
	return &MessageEcho{
		BaseEvent: eventbus.BaseEvent{
			ID:        uuid.New().String(),
			Name:      MessageEchoEventType,
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
