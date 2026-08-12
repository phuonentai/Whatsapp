package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	billingServices "github.com/moasq/go-b2b-starter/internal/modules/billing/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/tickets/domain"
	llmdomain "github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
)

// ErrAiCreditsExhausted is returned when the organization has a credit cap
// with no remaining AI credits; the handler maps it to HTTP 402
// ai_credits_exhausted.
var ErrAiCreditsExhausted = errors.New("créditos de IA agotados")

// TriageResult is the AI-drafted internal note plus a validated priority
// suggestion. The priority is nil when the model returns a value outside the
// ticket domain's valid set — the note is still returned (design D4).
type TriageResult struct {
	Note     string
	Priority *domain.TicketPriority
}

// AITriageService drafts an internal note and suggests a priority for a
// stored ticket through the metered, credit-gated LLM pipeline. It never
// mutates the ticket: the note is a draft, the priority a suggestion.
type AITriageService struct {
	llm     llmdomain.LLMClient
	billing billingServices.BillingService
	repo    domain.TicketRepository
}

// NewAITriageService mirrors the audience builder's constructor injection:
// the LLM client (metered by the platform) plus the billing service for the
// credit gate, over the org-scoped ticket repository.
func NewAITriageService(
	llm llmdomain.LLMClient,
	billing billingServices.BillingService,
	repo domain.TicketRepository,
) *AITriageService {
	return &AITriageService{llm: llm, billing: billing, repo: repo}
}

// triageSystemPrompt is the fixed Spanish system prompt. The ticket title and
// description are appended as user content (never interpolated, so a `%` in
// the description cannot corrupt formatting).
const triageSystemPrompt = `Eres un asistente de triage de tickets de soporte para una PYME colombiana.

Redacta una nota interna breve (máximo 120 palabras, en español) que resuma el problema del cliente y el contexto útil para el agente que lo atenderá, y sugiere una prioridad.

Responde SOLO con un objeto JSON, sin texto adicional y sin bloques de código:
{"note": "<nota interna>", "priority": "<alta|media|baja>"}

Reglas:
1. La nota debe ser profesional, objetiva y no incluir datos personales innecesarios.
2. La prioridad debe ser una de: alta, media, baja.
3. Usa "alta" solo para problemas urgentes o bloqueantes; de lo contrario usa "media" o "baja".
4. Si no hay suficiente contexto, usa "media" y resume lo que se sabe.`

// Triage runs the credit gate, loads the org-scoped ticket, and performs one
// metered LLM completion. Sentinel errors: ErrAiCreditsExhausted when the org
// is capped with no remaining credits, and domain.ErrTicketNotFound for a
// missing or foreign-org ticket. Ledger-read failures fail open (warn + go),
// matching the agent analysis semantics (design D3).
func (s *AITriageService) Triage(ctx context.Context, orgID, ticketID int32) (*TriageResult, error) {
	status, err := s.billing.GetAiUsageStatus(ctx, orgID)
	if err != nil {
		// Fail-open: a ledger outage must not block triage. The warn uses the
		// stdlib logger to keep the constructor dependency-free (design D2).
		log.Printf("ai usage status unavailable, proceeding fail-open: org=%d err=%v", orgID, err)
	} else if status != nil && status.CreditsMax > 0 && status.CreditsRemaining <= 0 {
		return nil, ErrAiCreditsExhausted
	}

	ticket, err := s.repo.GetByID(ctx, orgID, ticketID)
	if err != nil {
		return nil, err
	}
	if ticket == nil || ticket.OrganizationID != orgID {
		return nil, domain.ErrTicketNotFound
	}

	userPrompt := "Título del ticket: " + ticket.Title
	if desc := strings.TrimSpace(ticket.Description); desc != "" {
		userPrompt += "\nDescripción: " + desc
	}

	// Metered call, org-tagged (the platform client records token usage from
	// the org id in the context).
	ctx = llmdomain.WithOrgID(ctx, orgID)
	resp, err := s.llm.Complete(ctx, llmdomain.CompletionRequest{
		Prompt: triageSystemPrompt + "\n\n" + userPrompt,
	})
	if err != nil {
		return nil, fmt.Errorf("triage failed: %w", err)
	}

	result, err := parseTriageJSON(resp.Text)
	if err != nil {
		return nil, fmt.Errorf("triage response unparsable: %w", err)
	}
	return result, nil
}

// parseTriageJSON extracts the triage object from an LLM response, tolerating
// markdown code fences (same pattern as the audience builder's parser).
func parseTriageJSON(text string) (*TriageResult, error) {
	trimmed := strings.TrimSpace(text)
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	trimmed = strings.TrimSpace(trimmed)

	var raw struct {
		Note     string `json:"note"`
		Priority string `json:"priority"`
	}
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return nil, err
	}
	if strings.TrimSpace(raw.Note) == "" {
		return nil, fmt.Errorf("empty triage note")
	}

	result := &TriageResult{Note: strings.TrimSpace(raw.Note)}
	if p, ok := normalizePriority(raw.Priority); ok {
		result.Priority = &p
	}
	return result, nil
}

// normalizePriority maps the model's Spanish labels (and the English aliases)
// onto the ticket domain's valid set. Invalid or missing values yield
// ok=false so the priority is dropped while the note is still returned.
func normalizePriority(raw string) (domain.TicketPriority, bool) {
	p := domain.TicketPriority(strings.ToLower(strings.TrimSpace(raw)))
	switch p {
	case "alta", "alto", "high":
		p = domain.PriorityHigh
	case "media", "medio", "normal":
		p = domain.PriorityNormal
	case "baja", "bajo", "low":
		p = domain.PriorityLow
	}
	if !p.IsValid() {
		return "", false
	}
	return p, true
}
