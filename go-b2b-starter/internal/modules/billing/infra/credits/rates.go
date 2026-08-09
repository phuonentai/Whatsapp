package credits

import "math"

// ModelRate defines the credit cost per 1,000 tokens for a model.
// 1 credit is an abstract billing unit; rates are product-tunable constants.
type ModelRate struct {
	InputPer1K  float64
	OutputPer1K float64
}

// EmbeddingPer1K is the credit cost per 1,000 embedding tokens.
const EmbeddingPer1K = 0.0005

// defaultRate applies to unknown models (documented fallback).
var defaultRate = ModelRate{InputPer1K: 0.001, OutputPer1K: 0.002}

// modelRates maps known model names to their credit rates.
var modelRates = map[string]ModelRate{
	"gpt-5-mini":            {InputPer1K: 0.001, OutputPer1K: 0.002},
	"gpt-4.1-mini":          {InputPer1K: 0.001, OutputPer1K: 0.002},
	"gpt-4o-mini":           {InputPer1K: 0.001, OutputPer1K: 0.002},
	"text-embedding-3-small": {InputPer1K: 0, OutputPer1K: 0},
}

// RateFor returns the credit rate for a model, falling back to defaultRate.
func RateFor(model string) ModelRate {
	if rate, ok := modelRates[model]; ok {
		return rate
	}
	return defaultRate
}

// CreditsFor converts token consumption into credits for a model.
// Credits are rounded up so any consumption costs at least a measurable unit.
func CreditsFor(model string, inputTokens, outputTokens, embeddingTokens int64) int32 {
	rate := RateFor(model)

	credits := float64(inputTokens)/1000*rate.InputPer1K +
		float64(outputTokens)/1000*rate.OutputPer1K +
		float64(embeddingTokens)/1000*EmbeddingPer1K

	return int32(math.Ceil(credits))
}
