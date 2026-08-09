package ai

import (
	"context"

	"github.com/moasq/go-b2b-starter/internal/modules/cognitive/domain"
	llmdomain "github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
)

type openAIAssistantProvider struct {
	llmClient llmdomain.LLMClient
}

// NewAssistantProvider creates an AssistantProvider backed by OpenAI
func NewAssistantProvider(llmClient llmdomain.LLMClient) domain.AssistantProvider {
	return &openAIAssistantProvider{llmClient: llmClient}
}

func (p *openAIAssistantProvider) GenerateResponse(ctx context.Context, prompt string) (*domain.AssistantResponse, error) {
	req := llmdomain.CompletionRequest{Prompt: prompt}
	resp, err := p.llmClient.Complete(ctx, req)
	if err != nil {
		return nil, err
	}
	return &domain.AssistantResponse{
		Content:    resp.Text,
		TokensUsed: resp.TokensUsed,
	}, nil
}

func (p *openAIAssistantProvider) GenerateResponseStream(ctx context.Context, prompt string, emit func(domain.StreamEvent) error) (*domain.AssistantResponse, error) {
	req := llmdomain.CompletionRequest{Prompt: prompt}
	resp, err := p.llmClient.CompleteStream(ctx, req, func(chunk llmdomain.StreamChunk) error {
		if chunk.Done {
			return emit(domain.StreamEvent{Done: true})
		}
		return emit(domain.StreamEvent{Content: chunk.Content})
	})
	if err != nil {
		// Fail the stream even when the callback has not yet emitted a done
		// event so the SSE consumer always terminates.
		_ = emit(domain.StreamEvent{Done: true})
		return nil, err
	}
	return &domain.AssistantResponse{
		Content:    resp.Text,
		TokensUsed: resp.TokensUsed,
	}, nil
}
