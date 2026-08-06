package domain

import "time"

type ConversationStatus string

const (
	ConversationStatusActive   ConversationStatus = "active"
	ConversationStatusClosed   ConversationStatus = "closed"
	ConversationStatusArchived ConversationStatus = "archived"
)

type Conversation struct {
	ID             int32                  `json:"id"`
	OrganizationID int32                  `json:"organization_id"`
	ContactID      int32                  `json:"contact_id"`
	Status         ConversationStatus     `json:"status"`
	LastMessageAt  *time.Time             `json:"last_message_at,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

func (c *Conversation) IsActive() bool {
	return c.Status == ConversationStatusActive
}

func (c *Conversation) IsWithin24HourWindow() bool {
	if c.LastMessageAt == nil {
		return false
	}
	return time.Since(*c.LastMessageAt) < 24*time.Hour
}

type ConversationWithContact struct {
	Conversation
	ContactPhone       string `json:"contact_phone"`
	ContactDisplayName string `json:"contact_display_name"`
}
