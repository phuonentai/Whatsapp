package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
	billingServices "github.com/moasq/go-b2b-starter/internal/modules/billing/app/services"
	llmdomain "github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// draftingService drafts one personalized Spanish inquiry message per
// supplier through the metered LLM client (D3). Credits are checked before
// every call; exhaustion escalates without any unmetered invocation.
type draftingService struct {
	llm     llmdomain.LLMClient
	billing billingServices.BillingService
	audit   domain.AuditRepository
	metrics MetricsSink
	log     logger.Logger
}

// NewDraftingService builds the inquiry drafting service.
func NewDraftingService(
	llm llmdomain.LLMClient,
	billing billingServices.BillingService,
	audit domain.AuditRepository,
	metrics MetricsSink,
	log logger.Logger,
) DraftingService {
	if metrics == nil {
		metrics = noopMetrics{}
	}
	return &draftingService{llm: llm, billing: billing, audit: audit, metrics: metrics, log: log}
}

// DraftForSupplier builds the Spanish prompt (supplier display name greeting,
// product name × quantity, availability/price/lead-time asks) and runs exactly
// one metered LLM call. The prompt contains no contact PII beyond the supplier
// display name (NIT persona jurídica business identity, D11); documents and
// phones are never included. The context carries the org ID so the metered
// client records tokens into the ai-usage ledger.
func (s *draftingService) DraftForSupplier(ctx context.Context, orgID int32, supplier *domain.Supplier, displayName string, products []*domain.Product, quantities map[int32]float64) (string, error) {
	s.metrics.Inc(MetricDraftAttempt, map[string]string{"org": itoa(orgID)})

	if creditsExhausted(ctx, s.billing, orgID) {
		return "", domain.ErrCreditsExhausted
	}

	prompt := buildDraftPrompt(supplier, displayName, products, quantities)

	resp, err := s.llm.Complete(llmdomain.WithOrgID(ctx, orgID), llmdomain.CompletionRequest{
		Prompt: prompt,
	})
	if err != nil {
		return "", fmt.Errorf("draft completion: %w", err)
	}

	msg, err := parseDraftMessage(resp.Text)
	if err != nil {
		_ = s.recordMalformed(ctx, orgID, "draft", err)
		return "", domain.ErrMalformedLLMResponse
	}
	return msg, nil
}

func (s *draftingService) recordMalformed(ctx context.Context, orgID int32, entity string, cause error) error {
	return s.audit.Record(ctx, domain.AuditEntry{
		OrganizationID: orgID,
		EntityType:     entity,
		Action:         "llm_malformed",
		Decision:       "skip",
		Reason:         strPtr2("malformed_llm_json"),
		Metadata:       map[string]any{"error": cause.Error()},
	})
}

// buildDraftPrompt renders the Spanish drafting prompt. No documents, phones,
// or addresses are included — only the business display name (D11 allowlist).
func buildDraftPrompt(supplier *domain.Supplier, displayName string, products []*domain.Product, quantities map[int32]float64) string {
	var b strings.Builder
	b.WriteString("Eres un asistente de compras de una tienda colombiana sin inventario (venta sin inventario). ")
	b.WriteString("Redacta UN mensaje de WhatsApp en español, cortés y directo, para consultar disponibilidad, ")
	b.WriteString("precio y tiempo de entrega a un proveedor.\n\n")
	if displayName == "" {
		displayName = supplier.NIT
	}
	b.WriteString(fmt.Sprintf("Proveedor (nombre comercial): %s\n", displayName))
	b.WriteString("Productos solicitados:\n")
	for _, p := range products {
		qty := quantities[p.ID]
		if qty <= 0 {
			qty = 1
		}
		b.WriteString(fmt.Sprintf("- %s (SKU %s): %s\n", p.Name, p.SKU, formatQty(qty, p.Unit)))
	}
	b.WriteString("\nEl mensaje debe: saludar al proveedor, listar los productos con sus cantidades, ")
	b.WriteString("pedir disponibilidad, precio unitario y tiempo de entrega. ")
	b.WriteString("Debe ser un único mensaje de WhatsApp, en español neutro de Colombia, sin emojis. ")
	b.WriteString("Responde ÚNICAMENTE con JSON válido: {\"message\": \"<texto del mensaje>\"}. No agregues texto fuera del JSON.")
	return b.String()
}

func parseDraftMessage(text string) (string, error) {
	var out struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		// tolerate a fenced JSON block
		if start := strings.Index(text, "{"); start >= 0 {
			if end := strings.LastIndex(text, "}"); end > start {
				if err2 := json.Unmarshal([]byte(text[start:end+1]), &out); err2 == nil && out.Message != "" {
					return out.Message, nil
				}
			}
		}
		return "", err
	}
	if strings.TrimSpace(out.Message) == "" {
		return "", fmt.Errorf("empty drafted message")
	}
	return out.Message, nil
}

func formatQty(qty float64, unit string) string {
	if unit == "" {
		unit = "unidad(es)"
	}
	return fmt.Sprintf("%g %s", qty, unit)
}

func itoa(v int32) string {
	return fmt.Sprintf("%d", v)
}

func strPtr2(s string) *string { return &s }
