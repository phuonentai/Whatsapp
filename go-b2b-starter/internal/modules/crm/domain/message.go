package domain

import "time"

type MessageDirection string

const (
	MessageDirectionInbound  MessageDirection = "inbound"
	MessageDirectionOutbound MessageDirection = "outbound"
)

type MessageType string

const (
	MessageTypeText        MessageType = "text"
	MessageTypeImage       MessageType = "image"
	MessageTypeVideo       MessageType = "video"
	MessageTypeAudio       MessageType = "audio"
	MessageTypeDocument    MessageType = "document"
	MessageTypeLocation    MessageType = "location"
	MessageTypeInteractive MessageType = "interactive"
	MessageTypeButton      MessageType = "button"
	MessageTypeSticker     MessageType = "sticker"
	MessageTypeOrder       MessageType = "order"
	MessageTypeSystem      MessageType = "system"
)

type Message struct {
	ID                int32                  `json:"id"`
	OrganizationID    int32                  `json:"organization_id"`
	ConversationID    int32                  `json:"conversation_id"`
	ContactID         int32                  `json:"contact_id"`
	Channel           string                 `json:"channel"`
	ProviderMessageID string                 `json:"provider_message_id,omitempty"`
	Direction         MessageDirection       `json:"direction"`
	MessageType       MessageType            `json:"message_type"`
	Content           string                 `json:"content,omitempty"`
	Status            string                 `json:"status"`
	MessageData       map[string]interface{} `json:"message_data,omitempty"`
	ChatTimestamp     *time.Time             `json:"chat_timestamp,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

func (m *Message) Validate() error {
	if m.OrganizationID == 0 {
		return ErrContactOrganizationRequired
	}
	if m.ConversationID == 0 {
		return ErrConversationNotFound
	}
	return nil
}
