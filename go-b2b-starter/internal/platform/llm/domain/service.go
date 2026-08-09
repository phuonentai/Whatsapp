package domain

import (
	"context"
)

type CompletionRequest struct {
	Prompt      string
	MaxTokens   *int
	Temperature *float32
}

type CompletionResponse struct {
	Text       string
	TokensUsed int
	Model      string
}

type EmbeddingRequest struct {
	Text  string
	Model string
}

type EmbeddingResponse struct {
	Embedding  []float64
	TokensUsed int
	Model      string
}

type StreamChunk struct {
	Content string
	Done    bool
}

type LLMService interface {
	Complete(ctx context.Context, request CompletionRequest) (*CompletionResponse, error)
	CompleteStream(ctx context.Context, request CompletionRequest, callback func(StreamChunk) error) (*CompletionResponse, error)
}

type LLMClient interface {
	LLMService
	// GenerateEmbedding returns the embedding vector plus the number of tokens
	// consumed (for usage metering).
	GenerateEmbedding(ctx context.Context, text string, model string) ([]float64, int, error)
}

// UsageEvent describes one unit of AI consumption to be recorded in the
// tenant usage ledger. Tokens are recorded as consumed; credit conversion is
// an implementation concern of the ledger (billing module).
type UsageEvent struct {
	OrganizationID  int32
	Feature         string
	Model           string
	TokensInput     int64
	TokensOutput    int64
	TokensEmbedding int64
	RequestID       string
}

// TokenLedger records AI token consumption for an organization.
// Implemented by the billing module; consumed by metered LLM clients.
// Implementations MUST NOT block AI responses on ledger failures.
type TokenLedger interface {
	RecordUsage(ctx context.Context, event UsageEvent) error
}

type orgIDKey struct{}

// WithOrgID attaches the organization ID to a context for ledger recording.
func WithOrgID(ctx context.Context, organizationID int32) context.Context {
	return context.WithValue(ctx, orgIDKey{}, organizationID)
}

// OrgIDFromContext returns the organization ID attached via WithOrgID.
func OrgIDFromContext(ctx context.Context) (int32, bool) {
	orgID, ok := ctx.Value(orgIDKey{}).(int32)
	return orgID, ok
}

// PiiFacts carries the known PII values to mask before a provider call.
// Masking is exact-value replacement; empty fields are ignored.
type PiiFacts struct {
	PhoneNumber     string
	DisplayName     string
	NumeroDocumento string
}

type piiFactsKey struct{}

// WithPiiFacts attaches contact PII facts to a context for provider masking.
func WithPiiFacts(ctx context.Context, facts PiiFacts) context.Context {
	return context.WithValue(ctx, piiFactsKey{}, facts)
}

// PiiFactsFromContext returns the facts attached via WithPiiFacts.
func PiiFactsFromContext(ctx context.Context) (PiiFacts, bool) {
	facts, ok := ctx.Value(piiFactsKey{}).(PiiFacts)
	return facts, ok
}