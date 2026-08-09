package infra

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

var errTestLLM = errors.New("llm down")

// mockLLMClient is a controllable LLMClient double.
type mockLLMClient struct {
	completeResp  *domain.CompletionResponse
	completeErr   error
	embedding     []float64
	embeddingToks int
	embeddingErr  error
	streamResp    *domain.CompletionResponse
	streamErr     error
}

func (m *mockLLMClient) Complete(ctx context.Context, request domain.CompletionRequest) (*domain.CompletionResponse, error) {
	return m.completeResp, m.completeErr
}

func (m *mockLLMClient) CompleteStream(ctx context.Context, request domain.CompletionRequest, callback func(domain.StreamChunk) error) (*domain.CompletionResponse, error) {
	return m.streamResp, m.streamErr
}

func (m *mockLLMClient) GenerateEmbedding(ctx context.Context, text string, model string) ([]float64, int, error) {
	return m.embedding, m.embeddingToks, m.embeddingErr
}

// mockLedger captures recorded usage events.
type mockLedger struct {
	events []domain.UsageEvent
	err    error
}

func (l *mockLedger) RecordUsage(ctx context.Context, event domain.UsageEvent) error {
	if l.err != nil {
		return l.err
	}
	l.events = append(l.events, event)
	return nil
}

type noopLogger struct{}

func (noopLogger) Debug(msg string, fields ...logger.Fields) {}
func (noopLogger) Info(msg string, fields ...logger.Fields)  {}
func (noopLogger) Warn(msg string, fields ...logger.Fields)  {}
func (noopLogger) Error(msg string, fields ...logger.Fields) {}
func (noopLogger) Fatal(msg string, fields ...logger.Fields) {}
func (noopLogger) WithFields(fields logger.Fields) logger.Logger { return noopLogger{} }

func TestMeteredLLMClient_SuccessRecordsUsage(t *testing.T) {
	inner := &mockLLMClient{completeResp: &domain.CompletionResponse{Text: "hi", TokensUsed: 42, Model: "gpt-5-mini"}}
	ledger := &mockLedger{}
	client := NewMeteredLLMClient(inner, ledger, noopLogger{})

	ctx := domain.WithOrgID(context.Background(), 7)
	resp, err := client.Complete(ctx, domain.CompletionRequest{Prompt: "hello"})
	require.NoError(t, err)
	assert.Equal(t, "hi", resp.Text)

	require.Len(t, ledger.events, 1)
	event := ledger.events[0]
	assert.Equal(t, int32(7), event.OrganizationID)
	assert.Equal(t, "completion", event.Feature)
	assert.Equal(t, "gpt-5-mini", event.Model)
	assert.Equal(t, int64(42), event.TokensOutput)
	assert.NotEmpty(t, event.RequestID)
}

func TestMeteredLLMClient_EmbeddingRecordsUsage(t *testing.T) {
	inner := &mockLLMClient{embedding: []float64{1, 2, 3}, embeddingToks: 25}
	ledger := &mockLedger{}
	client := NewMeteredLLMClient(inner, ledger, noopLogger{})

	ctx := domain.WithOrgID(context.Background(), 9)
	vec, toks, err := client.GenerateEmbedding(ctx, "text", "text-embedding-3-small")
	require.NoError(t, err)
	assert.Equal(t, []float64{1, 2, 3}, vec)
	assert.Equal(t, 25, toks)

	require.Len(t, ledger.events, 1)
	assert.Equal(t, int64(25), ledger.events[0].TokensEmbedding)
	assert.Equal(t, "text-embedding-3-small", ledger.events[0].Model)
}

func TestMeteredLLMClient_LLMErrorRecordsNothing(t *testing.T) {
	inner := &mockLLMClient{completeErr: errTestLLM}
	ledger := &mockLedger{}
	client := NewMeteredLLMClient(inner, ledger, noopLogger{})

	_, err := client.Complete(domain.WithOrgID(context.Background(), 1), domain.CompletionRequest{})
	require.Error(t, err)
	assert.Empty(t, ledger.events)
}

func TestMeteredLLMClient_LedgerErrorFailsOpen(t *testing.T) {
	inner := &mockLLMClient{completeResp: &domain.CompletionResponse{Text: "ok", TokensUsed: 10, Model: "gpt-5-mini"}}
	ledger := &mockLedger{err: errors.New("db down")}
	client := NewMeteredLLMClient(inner, ledger, noopLogger{})

	resp, err := client.Complete(domain.WithOrgID(context.Background(), 1), domain.CompletionRequest{})
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Text)
}

func TestMeteredLLMClient_NoOrgInContextSkipsRecording(t *testing.T) {
	inner := &mockLLMClient{completeResp: &domain.CompletionResponse{Text: "ok", TokensUsed: 10, Model: "gpt-5-mini"}}
	ledger := &mockLedger{}
	client := NewMeteredLLMClient(inner, ledger, noopLogger{})

	resp, err := client.Complete(context.Background(), domain.CompletionRequest{})
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Text)
	assert.Empty(t, ledger.events)
}

func TestMeteredLLMClient_StreamRecordsFinalUsage(t *testing.T) {
	inner := &mockLLMClient{streamResp: &domain.CompletionResponse{Text: "done", TokensUsed: 99, Model: "gpt-5-mini"}}
	ledger := &mockLedger{}
	client := NewMeteredLLMClient(inner, ledger, noopLogger{})

	resp, err := client.CompleteStream(domain.WithOrgID(context.Background(), 3), domain.CompletionRequest{}, nil)
	require.NoError(t, err)
	assert.Equal(t, "done", resp.Text)
	require.Len(t, ledger.events, 1)
	assert.Equal(t, int64(99), ledger.events[0].TokensOutput)
}
