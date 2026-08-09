package infra

import (
	"context"

	"github.com/google/uuid"
	"github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// meteredLLMClient decorates an LLMClient with tenant usage ledger recording.
// Recording is fail-open: ledger failures are logged and never block the AI
// response. Usage is only recorded after a successful LLM call.
type meteredLLMClient struct {
	inner  domain.LLMClient
	ledger domain.TokenLedger
	log    logger.Logger
}

// NewMeteredLLMClient wraps an LLMClient so every completion, stream, and
// embedding call records its consumed tokens into the provided TokenLedger.
func NewMeteredLLMClient(inner domain.LLMClient, ledger domain.TokenLedger, log logger.Logger) domain.LLMClient {
	return &meteredLLMClient{inner: inner, ledger: ledger, log: log}
}

func (m *meteredLLMClient) Complete(ctx context.Context, request domain.CompletionRequest) (*domain.CompletionResponse, error) {
	resp, err := m.inner.Complete(ctx, request)
	if err != nil {
		return nil, err
	}
	m.record(ctx, domain.UsageEvent{
		Feature:      "completion",
		Model:        resp.Model,
		TokensOutput: int64(resp.TokensUsed),
		RequestID:    uuid.NewString(),
	})
	return resp, nil
}

func (m *meteredLLMClient) CompleteStream(ctx context.Context, request domain.CompletionRequest, callback func(domain.StreamChunk) error) (*domain.CompletionResponse, error) {
	resp, err := m.inner.CompleteStream(ctx, request, callback)
	if err != nil {
		return nil, err
	}
	m.record(ctx, domain.UsageEvent{
		Feature:      "completion_stream",
		Model:        resp.Model,
		TokensOutput: int64(resp.TokensUsed),
		RequestID:    uuid.NewString(),
	})
	return resp, nil
}

func (m *meteredLLMClient) GenerateEmbedding(ctx context.Context, text string, model string) ([]float64, int, error) {
	embedding, tokensUsed, err := m.inner.GenerateEmbedding(ctx, text, model)
	if err != nil {
		return nil, 0, err
	}
	if model == "" {
		model = "text-embedding-3-small" // Mirror the OpenAI client's default
	}
	m.record(ctx, domain.UsageEvent{
		Feature:         "embedding",
		Model:           model,
		TokensEmbedding: int64(tokensUsed),
		RequestID:       uuid.NewString(),
	})
	return embedding, tokensUsed, nil
}

func (m *meteredLLMClient) record(ctx context.Context, event domain.UsageEvent) {
	orgID, ok := domain.OrgIDFromContext(ctx)
	if !ok {
		m.log.Debug("ai usage event skipped: no organization in context", map[string]any{
			"feature": event.Feature,
		})
		return
	}
	event.OrganizationID = orgID

	if err := m.ledger.RecordUsage(ctx, event); err != nil {
		// Fail-open: metering must never break AI responses.
		m.log.Error("failed to record ai usage", map[string]any{
			"organization_id": orgID,
			"feature":         event.Feature,
			"error":           err.Error(),
		})
	}
}
