package domain

import "time"

// Mode identifies the agent autonomy mode for an organization.
type Mode string

const (
	ModeCopilot   Mode = "copilot"
	ModeAutopilot Mode = "autopilot"
)

// Tone identifies the language register used for generated replies.
type Tone string

const (
	ToneFormal Tone = "formal"
	ToneCasual Tone = "casual"
)

// FlowStatus is the lifecycle state of a conversation flow.
type FlowStatus string

const (
	FlowStatusRunning       FlowStatus = "running"
	FlowStatusAwaitingHuman FlowStatus = "awaiting_human"
	FlowStatusSucceeded     FlowStatus = "succeeded"
	FlowStatusFailed        FlowStatus = "failed"
	FlowStatusCancelled     FlowStatus = "cancelled"
)

// SuggestionType identifies the kind of agent suggestion.
type SuggestionType string

const (
	SuggestionReply      SuggestionType = "reply"
	SuggestionEscalation SuggestionType = "escalation"
)

// SuggestionStatus is the approval lifecycle of a suggestion.
type SuggestionStatus string

const (
	SuggestionPending    SuggestionStatus = "pending"
	SuggestionApproved   SuggestionStatus = "approved"
	SuggestionRejected   SuggestionStatus = "rejected"
	SuggestionSuperseded SuggestionStatus = "superseded"
)

// SuggestionSource records how a suggestion was created.
type SuggestionSource string

const (
	SuggestionSourceCopilot           SuggestionSource = "copilot"
	SuggestionSourceAutopilotFallback SuggestionSource = "autopilot_fallback"
	SuggestionSourceEscalation        SuggestionSource = "escalation"
)

// ConsentStatus is the Ley 1581 consent state of a contact.
type ConsentStatus string

const (
	ConsentNone      ConsentStatus = "none"
	ConsentRequested ConsentStatus = "requested"
	ConsentGranted   ConsentStatus = "granted"
	ConsentWithdrawn ConsentStatus = "withdrawn"
)

// AgentDecision is the outcome of a governance evaluation.
type AgentDecision string

const (
	DecisionAllow AgentDecision = "allow"
	DecisionDeny  AgentDecision = "deny"
	DecisionSkip  AgentDecision = "skip"
)

// DefaultGuardrails returns the built-in never/escalate rule skeleton.
func DefaultGuardrails() Guardrails {
	cap := 10.0
	return Guardrails{
		Never: &NeverRules{
			MaxDiscountPercent: &cap,
			ForbiddenTerms:     []string{},
		},
		Escalate: &EscalateRules{
			Terms: []string{"abogado", "legal", "garantía", "demanda", "superintendencia"},
		},
	}
}

// NeverRules are deterministic deny rules checked against every draft.
type NeverRules struct {
	MaxDiscountPercent *float64  `json:"max_discount_percent,omitempty"`
	ForbiddenTerms     []string  `json:"forbidden_terms,omitempty"`
}

// EscalateRules define topics that must be escalated to a human.
type EscalateRules struct {
	Terms []string `json:"terms,omitempty"`
}

// Guardrails is the tenant-editable rule set stored as JSONB.
type Guardrails struct {
	Never    *NeverRules    `json:"never,omitempty"`
	Escalate *EscalateRules `json:"escalate,omitempty"`
}

// AgentSettings is the per-organization agent configuration.
type AgentSettings struct {
	ID               int32
	OrganizationID   int32
	Mode             Mode
	Tone             Tone
	BrandVoice       string
	AutopilotStart   string // "HH:MM"
	AutopilotEnd     string // "HH:MM"
	Timezone         string
	KillSwitch       bool
	MaxDailyMessages int32
	ConsentRequired  bool
	ConsentTemplate  string
	Guardrails       Guardrails
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// DefaultSettings returns the defaults for organizations without a row:
// copilot mode, formal tone, consent required, America/Bogota.
func DefaultSettings(orgID int32) AgentSettings {
	return AgentSettings{
		OrganizationID:  orgID,
		Mode:            ModeCopilot,
		Tone:            ToneFormal,
		Timezone:        "America/Bogota",
		ConsentRequired: true,
		Guardrails:      DefaultGuardrails(),
	}
}

// ConversationFlow is one agent run per conversation.
type ConversationFlow struct {
	ID             int32
	OrganizationID int32
	ConversationID int32
	ContactID      int32
	Status         FlowStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Suggestion is a pending/approved/rejected draft or escalation.
type Suggestion struct {
	ID                 int32
	OrganizationID     int32
	ConversationID     int32
	ContactID          int32
	FlowID             *int32
	Type               SuggestionType
	Body               string
	Metadata           map[string]any
	Status             SuggestionStatus
	Source             SuggestionSource
	ApprovedByMemberID string
	WhatsAppMessageID  string
	RequestID          string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// AgentAction is an append-only governance audit row.
type AgentAction struct {
	ID                 int32
	OrganizationID     int32
	FlowID             *int32
	Action             string
	Decision           AgentDecision
	PolicyInput        map[string]any
	Reasons            []string
	ApprovedByMemberID string
	WhatsAppMessageID  string
	RequestID          string
	CreatedAt          time.Time
}
