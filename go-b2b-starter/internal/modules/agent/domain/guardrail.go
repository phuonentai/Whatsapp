package domain

import (
	"context"
	"time"
)

// ContactFacts is the PII/consent snapshot of a contact used by guardrail
// evaluation and PII masking. Never carries credentials.
type ContactFacts struct {
	PhoneNumber     string
	DisplayName     string
	NumeroDocumento string
	ConsentStatus   ConsentStatus
}

// GuardrailInput is the pure snapshot evaluated before a side-effecting action.
type GuardrailInput struct {
	Action      string
	Draft       string
	Settings    AgentSettings
	Contact     ContactFacts
	SentToday   int64
	Autonomous  bool // true for autopilot sends, false for human approvals
	Approver    string
	Now         time.Time
}

// GuardrailDecision is the outcome of a guardrail evaluation.
type GuardrailDecision struct {
	Allowed bool
	Reasons []string
}

// GuardrailAction names the governable agent actions.
const (
	GuardrailActionSendMessage  = "send_message"
	GuardrailActionEscalate     = "escalate_human"
	GuardrailActionGenerateDraft = "generate_draft"
)

// GuardrailService evaluates tenant-defined rules before side effects.
// Implementations MUST be transport-free (no HTTP, no external policy engines).
// Fail-safe direction: evaluation errors produce Deny with a guardrail_error
// reason, never Allow.
type GuardrailService interface {
	Evaluate(ctx context.Context, orgID int32, input GuardrailInput) (GuardrailDecision, error)
}
