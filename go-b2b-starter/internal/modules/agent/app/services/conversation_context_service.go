package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/agent/domain"
	billingServices "github.com/moasq/go-b2b-starter/internal/modules/billing/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain/conversationscope"
	llmdomain "github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// contextWindow is the number of most-recent messages fed to the LLM for
// context generation. Context is cheap and windowed by design.
const contextWindow = 30

// contextCacheTTL is how long a generated context row is served without a
// regeneration check, even when its cursor is stale.
const contextCacheTTL = 5 * time.Minute

// contextSystemPrompt asks the LLM to summarize a conversation as strict JSON.
const contextSystemPrompt = `Eres un asistente que resume conversaciones de WhatsApp para un asesor de ventas.
Responde SOLO con JSON válido, sin texto adicional ni bloques de código:
{"summary": "Resumen breve de la conversación en español (máx 3 oraciones).", "detected_intent": "intención detectada en 2-4 palabras", "key_facts": ["hecho relevante 1", "hecho relevante 2"]}
No inventes datos que no estén en la conversación. No incluyas números de teléfono ni datos personales sensibles en los hechos clave.`

// conversationContextService generates, caches, and serves per-conversation
// AI context. All LLM calls are metered (ledger recording via the org-tagged
// context) and PII-masked via the decorator chain.
type conversationContextService struct {
	repo    domain.AgentRepository
	llm     llmdomain.LLMClient
	billing billingServices.BillingService
	log     loggerdomain.Logger
}

// NewConversationContextService creates the context service.
func NewConversationContextService(
	repo domain.AgentRepository,
	llm llmdomain.LLMClient,
	billing billingServices.BillingService,
	log loggerdomain.Logger,
) domain.ConversationContextService {
	return &conversationContextService{repo: repo, llm: llm, billing: billing, log: log}
}

// GetContext returns AI-derived context for a conversation, generating and
// caching it when the cached row is stale (new messages since the generation
// cursor) or expired. Consent-gated orgs get structural-only context until the
// contact grants consent; withdrawn consent always masks the read.
func (s *conversationContextService) GetContext(ctx context.Context, orgID, conversationID int32, scope conversationscope.Scope) (*domain.ConversationContext, error) {
	conv, err := s.repo.GetConversationRef(ctx, orgID, conversationID, scope)
	if err != nil {
		return nil, err
	}

	meta, err := s.repo.GetConversationContextMeta(ctx, orgID, conversationID, scope)
	if err != nil {
		return nil, err
	}

	// Consent evaluation (Ley 1581): org requires consent + contact not granted
	// → structural only; withdrawn consent always masks the read.
	contact, err := s.repo.GetContactRef(ctx, orgID, conv.ContactID)
	if err != nil {
		return nil, err
	}
	settings, err := s.repo.GetSettings(ctx, orgID)
	if err != nil {
		if !errors.Is(err, domain.ErrSettingsNotFound) {
			return nil, err
		}
		settings = &domain.AgentSettings{ConsentRequired: domain.DefaultSettings(orgID).ConsentRequired}
	}
	consentGated := settings.ConsentRequired && contact.ConsentStatus != domain.ConsentGranted
	withdrawn := contact.ConsentStatus == domain.ConsentWithdrawn

	// Serve cached row when fresh and cursor-matched, unless consent state
	// changed since generation.
	cached, err := s.repo.GetConversationContext(ctx, orgID, conversationID)
	if err == nil {
		fresh := time.Since(cached.GeneratedAt) <= contextCacheTTL
		if fresh && cached.SourceCursor == meta.LatestMessageID && cached.ConsentGated == consentGated && !withdrawn {
			cached.Status = domain.ContextStatusAvailable
			return cached, nil
		}
	}

	structural := &domain.ConversationContext{
		ConversationID: conversationID,
		SourceCursor:   meta.LatestMessageID,
		ConsentGated:   consentGated,
		Status:         domain.ContextStatusStructural,
		Channel:        meta.Channel,
		MessageCount:   meta.MessageCount,
		FirstMessageAt: meta.FirstMessageAt,
		LastMessageAt:  meta.LastMessageAt,
	}
	if withdrawn || consentGated || meta.MessageCount == 0 {
		return s.persistStructural(ctx, orgID, structural)
	}

	// Credit gate — fail closed on exhausted credits (no unmetered fallback).
	status, err := s.billing.GetAiUsageStatus(ctx, orgID)
	if err != nil {
		s.log.Warn("ai usage status unavailable, treating as exhausted", loggerdomain.Fields{
			"org_id": orgID, "error": err.Error()})
		structural.Status = domain.ContextStatusUnavailable
		return structural, nil
	}
	if status != nil && status.CreditsMax > 0 && status.CreditsRemaining <= 0 {
		structural.Status = domain.ContextStatusUnavailable
		return structural, nil
	}

	generated, err := s.generate(ctx, orgID, conversationID, contact, scope)
	if err != nil {
		s.log.Warn("conversation context generation failed", loggerdomain.Fields{
			"org_id": orgID, "conversation_id": conversationID, "error": err.Error()})
		structural.Status = domain.ContextStatusUnavailable
		return structural, nil
	}
	return generated, nil
}

// persistStructural stores (and returns) a structural-only context row so the
// next read short-circuits the same work.
func (s *conversationContextService) persistStructural(ctx context.Context, orgID int32, c *domain.ConversationContext) (*domain.ConversationContext, error) {
	if c.MessageCount == 0 && c.SourceCursor == 0 {
		// Nothing to cache yet — return the structural projection as-is.
		return c, nil
	}
	_, err := s.repo.UpsertConversationContext(ctx, orgID, &domain.ConversationContext{
		ConversationID: c.ConversationID,
		SourceCursor:   c.SourceCursor,
		ConsentGated:   c.ConsentGated,
	})
	if err != nil {
		s.log.Warn("failed to persist structural context", loggerdomain.Fields{
			"org_id": orgID, "conversation_id": c.ConversationID, "error": err.Error()})
	}
	return c, nil
}

// generate builds a context from the recent message window via the metered,
// PII-masked LLM client and caches the result.
func (s *conversationContextService) generate(ctx context.Context, orgID, conversationID int32, contact *domain.ContactRef, scope conversationscope.Scope) (*domain.ConversationContext, error) {
	messages, err := s.repo.ListRecentConversationMessages(ctx, orgID, conversationID, contextWindow, scope)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages to generate context from")
	}

	var b strings.Builder
	for i := len(messages) - 1; i >= 0; i-- {
		dir := "cliente"
		if messages[i].Direction == "outbound" {
			dir = "asesor"
		}
		b.WriteString(fmt.Sprintf("%s: %s\n", dir, messages[i].Content))
	}

	ctx = llmdomain.WithOrgID(ctx, orgID)
	ctx = llmdomain.WithPiiFacts(ctx, llmdomain.PiiFacts{
		PhoneNumber:     contact.PhoneNumber,
		DisplayName:     contact.DisplayName,
		NumeroDocumento: contact.NumeroDocumento,
	})

	resp, err := s.llm.Complete(ctx, llmdomain.CompletionRequest{
		Prompt: contextSystemPrompt + "\n\nConversación:\n" + b.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("context generation failed: %w", err)
	}

	parsed, err := parseContextJSON(resp.Text)
	if err != nil {
		s.log.Warn("context response unparsable, storing raw summary", loggerdomain.Fields{
			"org_id": orgID, "conversation_id": conversationID, "error": err.Error()})
		parsed = &parsedContext{Summary: strings.TrimSpace(resp.Text)}
	}

	meta, err := s.repo.GetConversationContextMeta(ctx, orgID, conversationID, scope)
	if err != nil {
		return nil, err
	}

	c := &domain.ConversationContext{
		ConversationID: conversationID,
		Summary:        parsed.Summary,
		KeyFacts:       parsed.KeyFacts,
		DetectedIntent: parsed.DetectedIntent,
		SourceCursor:   meta.LatestMessageID,
		ConsentGated:   false,
		Status:         domain.ContextStatusAvailable,
	}
	persisted, err := s.repo.UpsertConversationContext(ctx, orgID, c)
	if err != nil {
		return nil, err
	}
	persisted.Status = domain.ContextStatusAvailable
	return persisted, nil
}

type parsedContext struct {
	Summary        string   `json:"summary"`
	DetectedIntent string   `json:"detected_intent"`
	KeyFacts       []string `json:"key_facts"`
}

// parseContextJSON extracts the JSON object from an LLM response, tolerating
// markdown code fences.
func parseContextJSON(text string) (*parsedContext, error) {
	trimmed := strings.TrimSpace(text)
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	trimmed = strings.TrimSpace(trimmed)

	start := strings.Index(trimmed, "{")
	if start < 0 {
		return nil, fmt.Errorf("no JSON object in response")
	}
	end := strings.LastIndex(trimmed, "}")
	if end < start {
		return nil, fmt.Errorf("unbalanced JSON braces")
	}
	var out parsedContext
	if err := json.Unmarshal([]byte(trimmed[start:end+1]), &out); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if strings.TrimSpace(out.Summary) == "" {
		return nil, fmt.Errorf("empty summary")
	}
	return &out, nil
}
