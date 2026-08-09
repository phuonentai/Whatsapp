package infra

import (
	"context"
	"strings"

	"github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// piiMaskingClient decorates an LLMClient so contact PII (document numbers,
// phone numbers, display names) is replaced with placeholders before the
// payload reaches the third-party AI provider. Original values stay in local
// PostgreSQL; masking is fail-open (a masking error logs and proceeds
// unmasked, per the whatsapp-compliance spec).
type piiMaskingClient struct {
	inner domain.LLMClient
	log   logger.Logger
}

// NewPiiMaskingClient wraps an LLMClient with Ley 1581 PII redaction.
func NewPiiMaskingClient(inner domain.LLMClient, log logger.Logger) domain.LLMClient {
	return &piiMaskingClient{inner: inner, log: log}
}

func (p *piiMaskingClient) Complete(ctx context.Context, request domain.CompletionRequest) (*domain.CompletionResponse, error) {
	request.Prompt = p.mask(ctx, request.Prompt)
	return p.inner.Complete(ctx, request)
}

func (p *piiMaskingClient) CompleteStream(ctx context.Context, request domain.CompletionRequest, callback func(domain.StreamChunk) error) (*domain.CompletionResponse, error) {
	request.Prompt = p.mask(ctx, request.Prompt)
	return p.inner.CompleteStream(ctx, request, callback)
}

func (p *piiMaskingClient) GenerateEmbedding(ctx context.Context, text string, model string) ([]float64, int, error) {
	return p.inner.GenerateEmbedding(ctx, p.mask(ctx, text), model)
}

// mask replaces known PII values with placeholders. Fail-open: errors are
// logged and the original text is returned.
func (p *piiMaskingClient) mask(ctx context.Context, text string) string {
	facts, ok := domain.PiiFactsFromContext(ctx)
	if !ok || text == "" {
		return text
	}

	replacements := []struct {
		value string
		label string
	}{
		{facts.NumeroDocumento, "[DOCUMENTO]"},
		{facts.PhoneNumber, "[TELEFONO]"},
		{facts.DisplayName, "[NOMBRE]"},
	}

	masked := text
	for _, r := range replacements {
		if r.value == "" {
			continue
		}
		masked = strings.ReplaceAll(masked, r.value, r.label)
	}

	if masked != text {
		p.log.Debug("pii masking applied", map[string]any{"fields": len(replacements)})
	}
	return masked
}
