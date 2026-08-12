package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// TemplateStatus is the local lifecycle of a WhatsApp message template.
// Local status changes only via explicit user action (draft edits, submit) or
// Meta-sourced events (webhook status update, manual sync). Meta remains the
// runtime approval authority; local `approved` gates sends.
type TemplateStatus string

const (
	TemplateStatusDraft     TemplateStatus = "draft"
	TemplateStatusSubmitted TemplateStatus = "submitted"
	TemplateStatusApproved  TemplateStatus = "approved"
	TemplateStatusRejected  TemplateStatus = "rejected"
	TemplateStatusPaused    TemplateStatus = "paused"
)

// paramPattern matches {{N}} placeholders in a template body.
var paramPattern = regexp.MustCompile(`\{\{\s*([0-9]+)\s*\}\}`)

// Template is the org-scoped registry entity for WhatsApp message templates.
// It never carries credentials: access tokens live only in whatsapp_configs.
type Template struct {
	ID              int64
	OrganizationID  int32
	Name            string
	Category        string
	Language        string
	Body            string
	ParamCount      int
	Status          TemplateStatus
	MetaTemplateID  *string
	RejectionReason *string
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CountParams returns the number of {{N}} placeholders in body, using the
// highest placeholder index as the count (a body with {{1}} and {{2}} has
// param_count 2). Non-contiguous or zero-indexed placeholders still yield the
// maximum index, keeping local count aligned with Meta's component count.
func CountParams(body string) int {
	maxIdx := 0
	matches := paramPattern.FindAllStringSubmatch(body, -1)
	for _, m := range matches {
		var idx int
		if _, err := fmt.Sscanf(m[1], "%d", &idx); err == nil && idx > maxIdx {
			maxIdx = idx
		}
	}
	return maxIdx
}

// Validate enforces authoring invariants with Spanish-first messages.
// Uniqueness on (organization_id, name, language) is enforced at the
// repository level (unique constraint → ErrTemplateNameConflict).
func (t *Template) Validate() error {
	if strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("El nombre de la plantilla es obligatorio")
	}
	if strings.TrimSpace(t.Category) == "" {
		return fmt.Errorf("La categoría de la plantilla es obligatoria")
	}
	if strings.TrimSpace(t.Language) == "" {
		return fmt.Errorf("El idioma de la plantilla es obligatorio")
	}
	if strings.TrimSpace(t.Body) == "" {
		return fmt.Errorf("El cuerpo de la plantilla es obligatorio")
	}
	return nil
}

// CanTransitionTo reports whether the local state machine allows moving to
// target. Allowed transitions: draft → submitted, submitted → approved |
// rejected | paused, rejected → draft, paused → submitted. Draft edits and
// delete only apply to draft templates (enforced by the service); approved
// templates are paused rather than deleted.
func (t *Template) CanTransitionTo(target TemplateStatus) bool {
	switch t.Status {
	case TemplateStatusDraft:
		return target == TemplateStatusSubmitted
	case TemplateStatusSubmitted:
		return target == TemplateStatusApproved ||
			target == TemplateStatusRejected ||
			target == TemplateStatusPaused
	case TemplateStatusRejected:
		return target == TemplateStatusDraft
	case TemplateStatusPaused:
		return target == TemplateStatusSubmitted
	default:
		return false
	}
}

// Transition validates and applies a state change, returning an error for
// forbidden transitions.
func (t *Template) Transition(target TemplateStatus) error {
	if !t.CanTransitionTo(target) {
		return fmt.Errorf("%w: %s → %s", ErrTemplateInvalidTransition, t.Status, target)
	}
	t.Status = target
	return nil
}

// IsEditable reports whether draft-only operations (update/delete) apply.
func (t *Template) IsEditable() bool {
	return t.Status == TemplateStatusDraft
}
