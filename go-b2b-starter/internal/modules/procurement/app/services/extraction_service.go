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

// extractionService runs exactly one metered LLM extraction call per eligible
// supplier reply (D4). The prompt carries the masked reply content only — no
// PII — and returns the structured quote contract.
type extractionService struct {
	llm     llmdomain.LLMClient
	billing billingServices.BillingService
	metrics MetricsSink
	log     logger.Logger
}

// NewExtractionService builds the reply extraction service.
func NewExtractionService(
	llm llmdomain.LLMClient,
	billing billingServices.BillingService,
	metrics MetricsSink,
	log logger.Logger,
) ExtractionService {
	if metrics == nil {
		metrics = noopMetrics{}
	}
	return &extractionService{llm: llm, billing: billing, metrics: metrics, log: log}
}

// ExtractReply runs one metered call and parses the quote contract
// {"items": [...], "resumen": ..., "requiere_humano": ...}.
func (s *extractionService) ExtractReply(ctx context.Context, orgID int32, content string) (*domain.ExtractionResult, error) {
	s.metrics.Inc(MetricExtractionAttempt, map[string]string{"org": itoa(orgID)})

	if creditsExhausted(ctx, s.billing, orgID) {
		return nil, domain.ErrCreditsExhausted
	}

	prompt := buildExtractionPrompt(content)

	resp, err := s.llm.Complete(llmdomain.WithOrgID(ctx, orgID), llmdomain.CompletionRequest{
		Prompt: prompt,
	})
	if err != nil {
		return nil, fmt.Errorf("extraction completion: %w", err)
	}

	result, err := parseExtractionResult(resp.Text)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrMalformedLLMResponse, err)
	}
	return result, nil
}

// buildExtractionPrompt renders the Spanish extraction prompt over the
// (masked) reply content.
func buildExtractionPrompt(reply string) string {
	return "Extrae la cotización del siguiente mensaje de WhatsApp de un proveedor colombiano.\n" +
		"Mensaje:\n" + reply + "\n\n" +
		"Responde ÚNICAMENTE con JSON válido con este contrato exacto:\n" +
		`{"items":[{"product_name":"","sku":"","disponible":true,"precio_unitario":0,"moneda":"COP","cantidad_disponible":0,"tiempo_entrega":"","requiere_seguimiento":false}],"resumen":"","requiere_humano":false}` + "\n" +
		"Reglas: usa null/omitir campos desconocidos; moneda siempre 'COP'; si el proveedor menciona rangos de precio, " +
		"negociación, \"depende\", o hay ambigüedad, pon requiere_humano=true; no inventes precios."
}

// parseExtractionResult parses the LLM JSON, tolerating a fenced block.
func parseExtractionResult(text string) (*domain.ExtractionResult, error) {
	var out domain.ExtractionResult
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		if start := strings.Index(text, "{"); start >= 0 {
			if end := strings.LastIndex(text, "}"); end > start {
				if err2 := json.Unmarshal([]byte(text[start:end+1]), &out); err2 == nil {
					return &out, nil
				}
			}
		}
		return nil, err
	}
	if out.Items == nil {
		out.Items = []domain.ResponseItem{}
	}
	return &out, nil
}
