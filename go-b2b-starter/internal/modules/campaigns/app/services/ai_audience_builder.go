package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	billingServices "github.com/moasq/go-b2b-starter/internal/modules/billing/app/services"
	crmDomain "github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	llmdomain "github.com/moasq/go-b2b-starter/internal/platform/llm/domain"

	"github.com/moasq/go-b2b-starter/internal/modules/campaigns/domain"
)

const audienceSystemPrompt = `Eres un asistente de segmentación de contactos para campañas de WhatsApp de una PYME colombiana.

Convierte la descripción del usuario en una especificación de filtros JSON. Reglas:
1. Responde SOLO con un array JSON de filtros, sin texto adicional, sin bloques de código.
2. Cada filtro tiene la forma: {"field": "<campo>", "op": "<operador>", "value": <valor>}.
3. Todos los filtros se combinan con Y (AND).

Campos y operadores permitidos:
- {"field": "source", "op": "eq", "value": "whatsapp"} — origen del contacto
- {"field": "lead_status", "op": "eq", "value": "nuevo|contactado|calificado|descalificado|cliente"} — estado comercial
- {"field": "company_id", "op": "eq", "value": <entero>} — ID de la empresa
- {"field": "assigned_to", "op": "eq", "value": <entero>} — ID del responsable
- {"field": "tag_ids", "op": "any", "value": [<enteros>]} — etiquetas (usa SOLO ids de la lista de etiquetas del usuario)
- {"field": "recency_days", "op": "lte", "value": <entero>} — contactados en los últimos N días
- {"field": "search", "op": "contains", "value": "<texto>"} — búsqueda por nombre/correo/teléfono/documento

Etiquetas de esta organización (nombre -> id):
%s

Si no puedes expresar el pedido con estos campos, usa los más cercanos. Si el pedido no menciona etiquetas, no incluyas tag_ids. Nunca inventes ids de etiquetas.`

type aiAudienceBuilder struct {
	llm           llmdomain.LLMClient
	billing       billingServices.BillingService
	tagRepo       crmDomain.TagRepository
	evaluator     domain.SegmentEvaluator
}

func NewAudienceBuilder(
	llm llmdomain.LLMClient,
	billing billingServices.BillingService,
	tagRepo crmDomain.TagRepository,
	evaluator domain.SegmentEvaluator,
) AudienceBuilderService {
	return &aiAudienceBuilder{llm: llm, billing: billing, tagRepo: tagRepo, evaluator: evaluator}
}

func (b *aiAudienceBuilder) Build(ctx context.Context, orgID int32, naturalLanguage string) (*domain.AudienceBuildResult, error) {
	if strings.TrimSpace(naturalLanguage) == "" {
		return nil, fmt.Errorf("%w: describe la audiencia que quieres", domain.ErrInvalidFilterSpec)
	}

	// Fail-closed on exhausted credits (same semantics as the agent pipeline).
	status, err := b.billing.GetAiUsageStatus(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to check ai usage status: %w", err)
	}
	if status != nil && status.CreditsMax > 0 && status.CreditsRemaining <= 0 {
		return nil, domain.ErrAiCreditsExhausted
	}

	tagDictionary, err := b.tagDictionary(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// Metered call, org-tagged. Prompt contains only the user's text plus the
	// field/op dictionary and tag names — zero contact PII.
	ctx = llmdomain.WithOrgID(ctx, orgID)
	resp, err := b.llm.Complete(ctx, llmdomain.CompletionRequest{
		Prompt: fmt.Sprintf(audienceSystemPrompt, tagDictionary) + "\n\nPedido: " + naturalLanguage,
	})
	if err != nil {
		return nil, fmt.Errorf("audience build failed: %w", err)
	}

	spec, err := parseFilterSpecJSON(resp.Text)
	if err != nil {
		return nil, fmt.Errorf("%w: la IA no pudo construir filtros válidos", domain.ErrInvalidFilterSpec)
	}

	// Same whitelist validation as manual segment CRUD.
	if err := domain.ValidateFilterSpec(spec); err != nil {
		return nil, err
	}

	if err := verifyTagIDs(ctx, b.tagRepo, orgID, spec); err != nil {
		return nil, err
	}

	preview, err := b.evaluator.Count(ctx, orgID, spec)
	if err != nil {
		return nil, err
	}

	return &domain.AudienceBuildResult{FilterSpec: spec, Preview: preview}, nil
}

// tagDictionary renders "name (id)" lines for the org's tags so the LLM can
// only reference real tag ids.
func (b *aiAudienceBuilder) tagDictionary(ctx context.Context, orgID int32) (string, error) {
	tags, err := b.tagRepo.List(ctx, orgID)
	if err != nil {
		return "", fmt.Errorf("failed to list tags: %w", err)
	}
	if len(tags) == 0 {
		return "(sin etiquetas)", nil
	}
	lines := make([]string, 0, len(tags))
	for _, t := range tags {
		lines = append(lines, fmt.Sprintf("%s (%d)", t.Nombre, t.ID))
	}
	return strings.Join(lines, ", "), nil
}

// verifyTagIDs rejects tag ids that do not belong to the organization.
// Shared by manual segment CRUD and the AI audience builder.
func verifyTagIDs(ctx context.Context, tagRepo crmDomain.TagRepository, orgID int32, spec []domain.Filter) error {
	tags, err := tagRepo.List(ctx, orgID)
	if err != nil {
		return fmt.Errorf("failed to list tags: %w", err)
	}
	valid := make(map[int32]bool, len(tags))
	for _, t := range tags {
		valid[t.ID] = true
	}
	for _, f := range spec {
		if f.Field != domain.FieldTagIDs {
			continue
		}
		arr, ok := toIntArray(f.Value)
		if !ok {
			continue
		}
		for _, id := range arr {
			if !valid[id] {
				return fmt.Errorf("%w: la etiqueta %d no existe en tu organización", domain.ErrInvalidFilterSpec, id)
			}
		}
	}
	return nil
}

// parseFilterSpecJSON extracts a JSON array from an LLM response, tolerating
// markdown code fences.
func parseFilterSpecJSON(text string) ([]domain.Filter, error) {
	trimmed := strings.TrimSpace(text)
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	trimmed = strings.TrimSpace(trimmed)

	var spec []domain.Filter
	if err := json.Unmarshal([]byte(trimmed), &spec); err != nil {
		return nil, err
	}
	if len(spec) == 0 {
		return nil, fmt.Errorf("empty filter spec")
	}
	return spec, nil
}
