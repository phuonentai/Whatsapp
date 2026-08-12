package domain

import (
	"context"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain/conversationscope"
)

// ContextStatus describes the availability of AI-derived conversation context.
type ContextStatus string

const (
	// ContextStatusAvailable means the context was generated (or served from
	// cache) and contains AI analysis.
	ContextStatusAvailable ContextStatus = "available"
	// ContextStatusUnavailable means generation was skipped (e.g. AI credits
	// exhausted or an LLM failure) with no unmetered fallback; the frontend
	// renders the "assistant is learning" state.
	ContextStatusUnavailable ContextStatus = "unavailable"
	// ContextStatusStructural means only structural information (counts,
	// dates, channel) is returned because consent gating or an empty history
	// prevented AI analysis.
	ContextStatusStructural ContextStatus = "structural"
)

// ConversationContext is the cached AI-derived context for one conversation.
type ConversationContext struct {
	ConversationID int32
	Summary        string
	KeyFacts       []string
	DetectedIntent string
	SourceCursor   int64
	ConsentGated   bool
	GeneratedAt    time.Time
	UpdatedAt      time.Time
	Status         ContextStatus
	// Structural projection (populated for structural/unavailable reads).
	Channel        string
	MessageCount   int64
	FirstMessageAt *time.Time
	LastMessageAt  *time.Time
}

// ConversationContextService generates and serves per-conversation AI
// context. Transport-free: no Stytch SDK or HTTP imports (Clean Architecture).
// El contexto se acota por el scope del miembro solicitante
// (conversation-row-scoping, task 4.5): fuera de scope → ErrConversationNotFound.
type ConversationContextService interface {
	GetContext(ctx context.Context, orgID, conversationID int32, scope conversationscope.Scope) (*ConversationContext, error)
}
