package infra

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
)

// capturingClient records the last prompt/text it received.
type capturingClient struct {
	lastPrompt string
	lastText   string
}

func (c *capturingClient) Complete(ctx context.Context, request domain.CompletionRequest) (*domain.CompletionResponse, error) {
	c.lastPrompt = request.Prompt
	return &domain.CompletionResponse{Text: "ok", TokensUsed: 1, Model: "gpt-5-mini"}, nil
}

func (c *capturingClient) CompleteStream(ctx context.Context, request domain.CompletionRequest, callback func(domain.StreamChunk) error) (*domain.CompletionResponse, error) {
	c.lastPrompt = request.Prompt
	return &domain.CompletionResponse{Text: "ok", TokensUsed: 1, Model: "gpt-5-mini"}, nil
}

func (c *capturingClient) GenerateEmbedding(ctx context.Context, text string, model string) ([]float64, int, error) {
	c.lastText = text
	return []float64{1}, 1, nil
}

func piiCtx() context.Context {
	return domain.WithPiiFacts(context.Background(), domain.PiiFacts{
		PhoneNumber:     "+573001234567",
		DisplayName:     "Juan Pérez",
		NumeroDocumento: "CC 123456789",
	})
}

func TestPiiMaskingClient_CompleteMasksKnownPII(t *testing.T) {
	inner := &capturingClient{}
	client := NewPiiMaskingClient(inner, noopLogger{})

	_, err := client.Complete(piiCtx(), domain.CompletionRequest{
		Prompt: "El contacto Juan Pérez con CC 123456789 y +573001234567 pregunta por el plan.",
	})
	require.NoError(t, err)

	assert.True(t, strings.Contains(inner.lastPrompt, "[NOMBRE]"), "name must be masked")
	assert.True(t, strings.Contains(inner.lastPrompt, "[DOCUMENTO]"), "document must be masked")
	assert.True(t, strings.Contains(inner.lastPrompt, "[TELEFONO]"), "phone must be masked")
	assert.False(t, strings.Contains(inner.lastPrompt, "Juan Pérez"), "raw name must not reach provider")
	assert.False(t, strings.Contains(inner.lastPrompt, "123456789"), "raw document must not reach provider")
	assert.False(t, strings.Contains(inner.lastPrompt, "3001234567"), "raw phone must not reach provider")
}

func TestPiiMaskingClient_StreamAndEmbeddingMasked(t *testing.T) {
	inner := &capturingClient{}
	client := NewPiiMaskingClient(inner, noopLogger{})

	_, err := client.CompleteStream(piiCtx(), domain.CompletionRequest{Prompt: "hola Juan Pérez"}, nil)
	require.NoError(t, err)
	assert.False(t, strings.Contains(inner.lastPrompt, "Juan Pérez"))

	_, _, err = client.GenerateEmbedding(piiCtx(), "embed Juan Pérez 123456789", "text-embedding-3-small")
	require.NoError(t, err)
	assert.False(t, strings.Contains(inner.lastText, "Juan Pérez"))
	assert.True(t, strings.Contains(inner.lastText, "[NOMBRE]"))
}

func TestPiiMaskingClient_NoFactsProceedsUnmasked(t *testing.T) {
	inner := &capturingClient{}
	client := NewPiiMaskingClient(inner, noopLogger{})

	_, err := client.Complete(context.Background(), domain.CompletionRequest{Prompt: "sin contexto"})
	require.NoError(t, err)
	assert.Equal(t, "sin contexto", inner.lastPrompt)
}
