package ai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/moasq/go-b2b-starter/internal/modules/cognitive/domain"
	llmdomain "github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
)

type mockLLMClient struct {
	chunks     []llmdomain.StreamChunk
	streamResp *llmdomain.CompletionResponse
	streamErr  error
	completeResp *llmdomain.CompletionResponse
	completeErr  error
	embedding  []float64
	embedTokens int
	embedErr    error
}

func (m *mockLLMClient) Complete(ctx context.Context, req llmdomain.CompletionRequest) (*llmdomain.CompletionResponse, error) {
	return m.completeResp, m.completeErr
}

func (m *mockLLMClient) CompleteStream(ctx context.Context, req llmdomain.CompletionRequest, callback func(llmdomain.StreamChunk) error) (*llmdomain.CompletionResponse, error) {
	for _, c := range m.chunks {
		if err := callback(c); err != nil {
			return nil, err
		}
	}
	return m.streamResp, m.streamErr
}

func (m *mockLLMClient) GenerateEmbedding(ctx context.Context, text string, model string) ([]float64, int, error) {
	return m.embedding, m.embedTokens, m.embedErr
}

func TestGenerateResponseStreamForwardsChunks(t *testing.T) {
	client := &mockLLMClient{
		chunks: []llmdomain.StreamChunk{
			{Content: "Hola ", Done: false},
			{Content: "mundo", Done: false},
			{Content: "", Done: true},
		},
		streamResp: &llmdomain.CompletionResponse{
			Text:       "Hola mundo",
			TokensUsed: 42,
			Model:      "gpt-test",
		},
	}

	provider := NewAssistantProvider(client)

	var received []domain.StreamEvent
	resp, err := provider.GenerateResponseStream(context.Background(), "prompt", func(ev domain.StreamEvent) error {
		received = append(received, ev)
		return nil
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Hola mundo", resp.Content)
	assert.Equal(t, 42, resp.TokensUsed)

	// Chunk forwarding: content chunks in order, then a done event.
	assert.Len(t, received, 3)
	assert.Equal(t, "Hola ", received[0].Content)
	assert.False(t, received[0].Done)
	assert.Equal(t, "mundo", received[1].Content)
	assert.False(t, received[1].Done)
	assert.True(t, received[2].Done)
}

func TestGenerateResponseStreamEmitsDoneOnError(t *testing.T) {
	client := &mockLLMClient{
		chunks:    []llmdomain.StreamChunk{{Content: "parcial", Done: false}},
		streamErr: assert.AnError,
	}

	provider := NewAssistantProvider(client)

	var received []domain.StreamEvent
	resp, err := provider.GenerateResponseStream(context.Background(), "prompt", func(ev domain.StreamEvent) error {
		received = append(received, ev)
		return nil
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
	// The done event must still be emitted so the SSE consumer terminates.
	assert.True(t, received[len(received)-1].Done)
}

func TestGenerateResponseNonStreaming(t *testing.T) {
	client := &mockLLMClient{
		completeResp: &llmdomain.CompletionResponse{
			Text:       "respuesta",
			TokensUsed: 7,
			Model:      "gpt-test",
		},
	}

	provider := NewAssistantProvider(client)
	resp, err := provider.GenerateResponse(context.Background(), "prompt")

	assert.NoError(t, err)
	assert.Equal(t, "respuesta", resp.Content)
	assert.Equal(t, 7, resp.TokensUsed)
}
