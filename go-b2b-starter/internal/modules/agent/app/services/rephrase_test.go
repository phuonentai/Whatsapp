package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	billingDomain "github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
	llmdomain "github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
)

// recordingLLM captures every completion call (context + request) so tests
// can assert metering attribution and that no LLM call happens on gated paths.
type recordingLLM struct {
	calls []llmdomain.CompletionRequest
	ctxs  []context.Context
	text  string
	err   error
}

func (m *recordingLLM) Complete(ctx context.Context, request llmdomain.CompletionRequest) (*llmdomain.CompletionResponse, error) {
	m.calls = append(m.calls, request)
	m.ctxs = append(m.ctxs, ctx)
	if m.err != nil {
		return nil, m.err
	}
	return &llmdomain.CompletionResponse{Text: m.text, TokensUsed: 10, Model: "gpt-test"}, nil
}

func (m *recordingLLM) CompleteStream(ctx context.Context, request llmdomain.CompletionRequest, callback func(llmdomain.StreamChunk) error) (*llmdomain.CompletionResponse, error) {
	return nil, errors.New("not used")
}

func (m *recordingLLM) GenerateEmbedding(ctx context.Context, text string, model string) ([]float64, int, error) {
	return nil, 0, errors.New("not used")
}

func TestRephraseTextEachModeReturnsTransformedText(t *testing.T) {
	for _, mode := range []string{"rephrase", "formal", "casual", "summarize"} {
		t.Run(mode, func(t *testing.T) {
			llm := &recordingLLM{text: "  Texto transformado por la IA.  "}
			svc := newTestService(
				newMockRepo(), &mockGuardrails{}, llm,
				&mockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 90}},
				&mockOutbound{},
			)

			out, err := svc.RephraseText(context.Background(), 42, "hola cuando llega mi pedido", mode)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out != "Texto transformado por la IA." {
				t.Fatalf("expected trimmed transformed text, got %q", out)
			}
			if len(llm.calls) != 1 {
				t.Fatalf("expected exactly one LLM call, got %d", len(llm.calls))
			}
			if !strings.HasPrefix(llm.calls[0].Prompt, rephraseSystemPrompt(mode)) {
				t.Fatalf("expected mode system prompt prefix, got %q", llm.calls[0].Prompt)
			}
			orgID, ok := llmdomain.OrgIDFromContext(llm.ctxs[0])
			if !ok || orgID != 42 {
				t.Fatalf("expected org-scoped metering context (org 42), got orgID=%d ok=%v", orgID, ok)
			}
		})
	}
}

func TestRephraseTextExhaustedCreditsSkipsLLMCall(t *testing.T) {
	llm := &recordingLLM{text: "no debería llamarse"}
	svc := newTestService(
		newMockRepo(), &mockGuardrails{}, llm,
		&mockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 0}},
		&mockOutbound{},
	)

	_, err := svc.RephraseText(context.Background(), 42, "hola", "rephrase")
	if !errors.Is(err, ErrAICreditsExhausted) {
		t.Fatalf("expected ErrAICreditsExhausted, got %v", err)
	}
	if len(llm.calls) != 0 {
		t.Fatalf("exhausted credits must not call the LLM, got %d calls", len(llm.calls))
	}
}

func TestRephraseTextLedgerFailureFailsOpen(t *testing.T) {
	llm := &recordingLLM{text: "texto transformado"}
	svc := newTestService(
		newMockRepo(), &mockGuardrails{}, llm,
		&mockBilling{err: errors.New("ledger down")},
		&mockOutbound{},
	)

	out, err := svc.RephraseText(context.Background(), 42, "hola", "casual")
	if err != nil {
		t.Fatalf("ledger failure must fail open, got %v", err)
	}
	if out != "texto transformado" {
		t.Fatalf("unexpected output %q", out)
	}
	if len(llm.calls) != 1 {
		t.Fatalf("expected LLM call despite ledger failure, got %d", len(llm.calls))
	}
}

func TestRephraseTextInvalidModeErrorsBeforeLLM(t *testing.T) {
	llm := &recordingLLM{text: "x"}
	svc := newTestService(
		newMockRepo(), &mockGuardrails{}, llm,
		&mockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 90}},
		&mockOutbound{},
	)

	_, err := svc.RephraseText(context.Background(), 42, "hola", "poetic")
	if err == nil {
		t.Fatal("invalid mode must error")
	}
	if len(llm.calls) != 0 {
		t.Fatalf("invalid mode must not call the LLM, got %d calls", len(llm.calls))
	}
}

func TestRephraseTextLLMFailureWrapsError(t *testing.T) {
	llm := &recordingLLM{err: errors.New("provider down")}
	svc := newTestService(
		newMockRepo(), &mockGuardrails{}, llm,
		&mockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 90}},
		&mockOutbound{},
	)

	_, err := svc.RephraseText(context.Background(), 42, "hola", "formal")
	if err == nil || !strings.Contains(err.Error(), "agent rephrase failed") {
		t.Fatalf("expected wrapped LLM error, got %v", err)
	}
}
