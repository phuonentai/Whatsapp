package services

import (
	"context"

	"github.com/moasq/go-b2b-starter/internal/modules/agent/domain"
	igEvents "github.com/moasq/go-b2b-starter/internal/modules/instagram/domain/events"
	whatsappEvents "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
)

// AgentService drives the agentic messaging assistant pipeline.
type AgentService interface {
	// HandleMessageReceived runs the pipeline for an inbound WhatsApp event
	// (analysis, consent, guardrails, copilot draft or autopilot send).
	HandleMessageReceived(ctx context.Context, event *whatsappEvents.MessageReceived) error

	// HandleInstagramMessageReceived runs the same pipeline for an inbound
	// Instagram DM (IG mid as provider message id).
	HandleInstagramMessageReceived(ctx context.Context, event *igEvents.MessageReceived) error

	// ApproveSuggestion sends a pending suggestion after guardrail evaluation.
	// The suggestion is sent only when the decision is allow; denials are
	// audited and the suggestion is rejected.
	ApproveSuggestion(ctx context.Context, orgID, suggestionID int32, editedBody, memberID string) (*domain.Suggestion, error)

	// RejectSuggestion marks a pending suggestion rejected (no send).
	RejectSuggestion(ctx context.Context, orgID, suggestionID int32) (*domain.Suggestion, error)

	// ListPendingSuggestions returns the org's pending queue.
	ListPendingSuggestions(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.Suggestion, error)

	// SeedPendingSuggestion inserts a pending reply suggestion for a
	// conversation without running the LLM pipeline (mock-auth test seeding).
	SeedPendingSuggestion(ctx context.Context, orgID, conversationID int32, body string) (*domain.Suggestion, error)

	// GetFlowDebug returns the latest flow for a conversation with its suggestions.
	GetFlowDebug(ctx context.Context, orgID, conversationID int32) (*FlowDebug, error)

	// GetSettings returns the org's agent settings (defaults on first read).
	GetSettings(ctx context.Context, orgID int32) (*domain.AgentSettings, error)

	// UpdateSettings validates and persists the org's agent settings.
	UpdateSettings(ctx context.Context, orgID int32, s *domain.AgentSettings) (*domain.AgentSettings, error)
}

// FlowDebug is the debug projection of a conversation flow.
type FlowDebug struct {
	Flow        *domain.ConversationFlow `json:"flow"`
	Suggestions []*domain.Suggestion     `json:"suggestions"`
}

// ComplianceService provides Ley 1581 controls.
type ComplianceService interface {
	// ExportContact returns the contact's data bundle for Habeas Data requests,
	// masking PII when consent is withdrawn.
	ExportContact(ctx context.Context, orgID, contactID int32) (*ExportBundle, error)

	// ForgetContact anonymizes the contact's PII and withdraws consent.
	ForgetContact(ctx context.Context, orgID, contactID int32) error
}

// ExportBundle is the structured data-portability payload.
type ExportBundle struct {
	Contact       *ContactExport        `json:"contact"`
	Conversations []*ConversationExport `json:"conversations"`
}

// ContactExport is the masked-or-raw contact profile projection.
type ContactExport struct {
	PhoneNumber     string `json:"phone_number"`
	DisplayName     string `json:"display_name,omitempty"`
	Email           string `json:"email,omitempty"`
	TipoDocumento   string `json:"tipo_documento,omitempty"`
	NumeroDocumento string `json:"numero_documento,omitempty"`
	ConsentStatus   string `json:"consent_status"`
	ConsentedAt     string `json:"consented_at,omitempty"`
}

// ConversationExport is one conversation with its messages.
type ConversationExport struct {
	ID       int32            `json:"id"`
	Status   string           `json:"status"`
	Messages []*MessageExport `json:"messages"`
}

// MessageExport is one message in an export bundle.
type MessageExport struct {
	Direction string `json:"direction"`
	Type      string `json:"message_type"`
	Content   string `json:"content"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}
