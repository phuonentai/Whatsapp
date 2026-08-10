package domain

import (
	"context"
	"time"
)

// Channel identifies the messaging provider a conversation belongs to.
const (
	ChannelWhatsapp  = "whatsapp"
	ChannelInstagram = "instagram"
)

// AgentRepository persists the agent pipeline state and compliance data.
// All methods are scoped by organization_id (Stytch-org FK pattern).
type AgentRepository interface {
	// Flows
	CreateFlow(ctx context.Context, orgID, conversationID, contactID int32) (*ConversationFlow, error)
	GetFlow(ctx context.Context, orgID, flowID int32) (*ConversationFlow, error)
	GetActiveFlowByConversation(ctx context.Context, orgID, conversationID int32) (*ConversationFlow, error)
	UpdateFlowStatus(ctx context.Context, orgID, flowID int32, status FlowStatus) (*ConversationFlow, error)

	// Settings
	GetSettings(ctx context.Context, orgID int32) (*AgentSettings, error)
	UpsertSettings(ctx context.Context, settings *AgentSettings) (*AgentSettings, error)

	// Suggestions
	InsertSuggestion(ctx context.Context, s *Suggestion) (*Suggestion, error)
	ListSuggestions(ctx context.Context, orgID int32, status SuggestionStatus, limit, offset int32) ([]*Suggestion, error)
	GetSuggestion(ctx context.Context, orgID, suggestionID int32) (*Suggestion, error)
	ApproveSuggestion(ctx context.Context, orgID, suggestionID int32, approvedByMember string) (*Suggestion, error)
	RejectSuggestion(ctx context.Context, orgID, suggestionID int32) (*Suggestion, error)
	SupersedePendingReplies(ctx context.Context, orgID, conversationID int32) error
	GetPendingSuggestionByMessage(ctx context.Context, orgID int32, whatsappMessageID string) (*Suggestion, error)

	// Audit
	InsertAction(ctx context.Context, record *AgentAction) (*AgentAction, error)

	// Usage
	CountMessagesSentToday(ctx context.Context, orgID int32, since time.Time) (int64, error)

	// Contact/conversation resolution (idempotent, mirrors CRM upsert patterns)
	ResolveContact(ctx context.Context, orgID int32, phoneNumber, displayName string, lastMessageAt time.Time) (*ContactRef, error)
	ResolveContactByIGUser(ctx context.Context, orgID int32, igUserID, displayName string, lastMessageAt time.Time) (*ContactRef, error)
	GetContactRef(ctx context.Context, orgID, contactID int32) (*ContactRef, error)
	ResolveConversation(ctx context.Context, orgID, contactID int32, channel string, lastMessageAt time.Time) (*ConversationRef, error)
	GetConversationRef(ctx context.Context, orgID, conversationID int32) (*ConversationRef, error)
	ListConversationsByContact(ctx context.Context, orgID, contactID int32) ([]*ConversationRef, error)
	ListMessagesByConversation(ctx context.Context, orgID, conversationID int32, limit, offset int32) ([]*MessageRef, error)

	// Compliance (Ley 1581)
	UpdateContactConsent(ctx context.Context, orgID, contactID int32, status ConsentStatus, consentedAt *time.Time) (*ContactRef, error)
	AnonymizeContact(ctx context.Context, orgID, contactID int32) error
}

// OutboundGateway sends an outbound WhatsApp message through the existing
// outbound seam (implemented by the CRM module's OutboundService). Kept in the
// agent domain so the pipeline has a transport-free abstraction for sending.
type OutboundGateway interface {
	SendMessage(ctx context.Context, orgID, conversationID int32, content string) error
}

// ContactRef is the minimal contact projection used by the agent pipeline.
type ContactRef struct {
	ID              int32
	OrganizationID  int32
	PhoneNumber     string
	DisplayName     string
	Email           string
	TipoDocumento   string
	NumeroDocumento string
	ConsentStatus   ConsentStatus
	ConsentedAt     *time.Time
}

// ConversationRef is the minimal conversation projection.
type ConversationRef struct {
	ID             int32
	OrganizationID int32
	ContactID      int32
	Status         string
	LastMessageAt  *time.Time
}

// MessageRef is the minimal message projection used for export bundles.
type MessageRef struct {
	ID                int32
	OrganizationID    int32
	ConversationID    int32
	ContactID         int32
	Direction         string
	MessageType       string
	Content           string
	Status            string
	ProviderMessageID string
	CreatedAt         time.Time
}
