package credits

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreditsFor_KnownModel(t *testing.T) {
	// gpt-5-mini: 0.001 credits / 1K input, 0.002 credits / 1K output
	assert.Equal(t, int32(0), CreditsFor("gpt-5-mini", 0, 0, 0))
	assert.Equal(t, int32(1), CreditsFor("gpt-5-mini", 1000, 0, 0))
	assert.Equal(t, int32(1), CreditsFor("gpt-5-mini", 0, 1000, 0))
	assert.Equal(t, int32(1), CreditsFor("gpt-5-mini", 1000, 1000, 2000))
	// Larger volumes accumulate linearly: 1M input tokens → 1 credit, 10M → 10
	assert.Equal(t, int32(1), CreditsFor("gpt-5-mini", 1000000, 0, 0))
	assert.Equal(t, int32(10), CreditsFor("gpt-5-mini", 10000000, 0, 0))
}

func TestCreditsFor_UnknownModelFallsBack(t *testing.T) {
	// default: 0.001 input, 0.002 output, 0.0005 embedding per 1K
	assert.Equal(t, int32(1), CreditsFor("my-custom-model", 1000, 0, 0))
	assert.Equal(t, int32(1), CreditsFor("my-custom-model", 0, 1000, 0))
	assert.Equal(t, int32(1), CreditsFor("my-custom-model", 1000, 1000, 2000))
	assert.Equal(t, int32(5), CreditsFor("my-custom-model", 5000000, 0, 0))
}

func TestCreditsFor_EmbeddingOnly(t *testing.T) {
	// text-embedding-3-small has zero completion rates; only the embedding
	// constant applies (0.0005 per 1K).
	assert.Equal(t, int32(1), CreditsFor("text-embedding-3-small", 0, 0, 2000))
	assert.Equal(t, int32(1), CreditsFor("text-embedding-3-small", 0, 0, 1001))
}

func TestCreditsFor_RoundsUp(t *testing.T) {
	// 1 token at default input rate → 0.000001 credits → ceil → 1
	assert.Equal(t, int32(1), CreditsFor("gpt-5-mini", 1, 0, 0))
}

func TestCreditsFor_ZeroTokens(t *testing.T) {
	assert.Equal(t, int32(0), CreditsFor("gpt-5-mini", 0, 0, 0))
	assert.Equal(t, int32(0), CreditsFor("unknown", 0, 0, 0))
}
