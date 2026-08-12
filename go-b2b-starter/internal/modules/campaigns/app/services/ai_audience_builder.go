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

const audienceSystemPrompt = `Eres un asistente de segmentación y redacción para campañas de WhatsApp de una PYME colombiana.

Convierte la descripción del usuario en un objeto JSON con dos campos:
- "filter_spec": un array de filtros (ver reglas abajo).
- "message_draft": un borrador de mensaje promocional de WhatsApp en español para esa audiencia.

Reglas del filter_spec:
1. Cada filtro tiene la forma: {"field": "<campo>", "op": "<operador>", "value": <valor>}.
2. Todos los filtros se combinan con Y (AND).

Campos y operadores permitidos:
- {"field": "source", "op": "eq", "value": "whatsapp"} — origen del contacto
- {"field": "lead_status", "op": "eq", "value": "nuevo|contactado|calificado|descalificado|cliente"} — estado comercial
- {"field": "company_id", "op": "eq", "value": <entero>} — ID de la empresa
- {"field": "assigned_to", "op": "eq", "value": <entero>} — ID del responsable
- {"field": "tag_ids", "op": "any", "value": [<enteros>]} — etiquetas (usa SOLO ids de la lista de etiquetas del usuario)
- {"field": "recency_days", "op": "lte", "value": <entero>} — contactados en los últimos N días
- {"field": "search", "op": "contains", "value": "<texto>"} — búsqueda por nombre/correo/teléfono/documento

Reglas del message_draft:
1. Redacta en español, de 1 a 3 oraciones, con un tono acorde a la descripción (formal, promocional o urgente según lo que pida el usuario).
2. Incluye una llamada a la acción (CTA) clara.
3. Cumple la Ley 1581: sin afirmaciones engañosas ni promesas exageradas; si corresponde, recuerda el consentimiento del destinatario.
4. NO uses marcadores de PII de contactos: nada de nombres, teléfonos ni documentos en el texto.
5. Si la descripción no da contexto suficiente para redactar el mensaje, deja "message_draft" vacío ("").

Etiquetas de esta organización (nombre -> id):
%s

Si no puedes expresar el pedido con estos campos, usa los más cercanos. Si el pedido no menciona etiquetas, no incluyas tag_ids. Nunca inventes ids de etiquetas.

Responde SOLO con un objeto JSON válido: {"filter_spec": [...], "message_draft": "..."}, sin texto adicional ni bloques de código.`

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

	spec, messageDraft, err := parseAIBuildResponse(resp.Text)
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

	return &domain.AudienceBuildResult{FilterSpec: spec, Preview: preview, MessageDraft: messageDraft}, nil
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

// parseAIBuildResponse extracts the filter spec (mandatory) and the optional
// message draft from an LLM response, tolerating markdown code fences.
//   - filter_spec: required; unparsable specs fail the call.
//   - message_draft: optional; unparsable or empty drafts are dropped and
//     never fail the call (per-field resilience, mirroring ticket triage).
//
// A bare filter-spec array (legacy contract) is still accepted as a spec
// with no message draft.
func parseAIBuildResponse(text string) ([]domain.Filter, string, error) {
	trimmed := strings.TrimSpace(text)
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	trimmed = strings.TrimSpace(trimmed)

	// Legacy contract: the model answered with a bare filter spec array.
	var rawSpec []domain.Filter
	if err := json.Unmarshal([]byte(trimmed), &rawSpec); err == nil {
		if len(rawSpec) == 0 {
			return nil, "", fmt.Errorf("empty filter spec")
		}
		return rawSpec, "", nil
	}

	var obj struct {
		FilterSpec   json.RawMessage `json:"filter_spec"`
		MessageDraft json.RawMessage `json:"message_draft"`
	}
	if err := json.Unmarshal([]byte(trimmed), &obj); err != nil {
		return nil, "", err
	}
	if len(obj.FilterSpec) == 0 {
		return nil, "", fmt.Errorf("missing filter_spec")
	}

	var spec []domain.Filter
	if err := json.Unmarshal(obj.FilterSpec, &spec); err != nil {
		return nil, "", err
	}
	if len(spec) == 0 {
		return nil, "", fmt.Errorf("empty filter spec")
	}

	// Optional message draft: drop on parse failure, keep the spec.
	var draft string
	if len(obj.MessageDraft) > 0 {
		if err := json.Unmarshal(obj.MessageDraft, &draft); err != nil {
			draft = ""
		}
	}
	return spec, strings.TrimSpace(draft), nil
}
