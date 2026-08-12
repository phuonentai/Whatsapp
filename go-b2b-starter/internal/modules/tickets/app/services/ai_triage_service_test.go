package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	billingServices "github.com/moasq/go-b2b-starter/internal/modules/billing/app/services"
	billingDomain "github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/tickets/domain"
	llmdomain "github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
)

// ---------- mocks ----------

// recordingLLM records the context and prompt of every completion so tests can
// assert metering wiring (org id attached via WithOrgID) and prompt content.
type recordingLLM struct {
	llmdomain.LLMClient
	text  string
	err   error
	ctxs  []context.Context
	calls []llmdomain.CompletionRequest
}

func (m *recordingLLM) Complete(ctx context.Context, req llmdomain.CompletionRequest) (*llmdomain.CompletionResponse, error) {
	m.ctxs = append(m.ctxs, ctx)
	m.calls = append(m.calls, req)
	if m.err != nil {
		return nil, m.err
	}
	return &llmdomain.CompletionResponse{Text: m.text, TokensUsed: 12, Model: "test-model"}, nil
}

type triageMockBilling struct {
	billingServices.BillingService
	status *billingDomain.AiUsageStatus
	err    error
}

func (m *triageMockBilling) GetAiUsageStatus(ctx context.Context, orgID int32) (*billingDomain.AiUsageStatus, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.status, nil
}

// newTriageHarness seeds a repo with ticket 1 (org 1) and ticket 2 (org 2),
// and returns the service wired with the given mocks.
func newTriageHarness(t *testing.T, llm llmdomain.LLMClient, billing billingServices.BillingService) (*AITriageService, *fakeTicketRepo) {
	t.Helper()
	repo := newFakeTicketRepo()
	_, err := repo.Create(context.Background(), &domain.Ticket{
		OrganizationID: 1, Title: "Problema con factura", Description: "No me llegó la factura de julio",
	})
	if err != nil {
		t.Fatalf("seed ticket 1: %v", err)
	}
	_, err = repo.Create(context.Background(), &domain.Ticket{
		OrganizationID: 2, Title: "Ticket ajeno", Description: "no debe verse",
	})
	if err != nil {
		t.Fatalf("seed ticket 2: %v", err)
	}
	return NewAITriageService(llm, billing, repo), repo
}

// ---------- service tests ----------

func TestAITriageHappyPath_ReturnsNoteAndValidPriority(t *testing.T) {
	llm := &recordingLLM{text: `{"note":"El cliente no recibió la factura de julio.","priority":"alta"}`}
	svc, _ := newTriageHarness(t, llm, &triageMockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 50}})

	result, err := svc.Triage(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("triage failed: %v", err)
	}
	if result.Note != "El cliente no recibió la factura de julio." {
		t.Fatalf("unexpected note: %q", result.Note)
	}
	if result.Priority == nil || *result.Priority != domain.PriorityHigh {
		t.Fatalf("expected high priority, got %v", result.Priority)
	}

	// Metered recording: org id attached to the completion context and the
	// prompt carries the stored title + description.
	if len(llm.ctxs) != 1 {
		t.Fatalf("expected 1 LLM call, got %d", len(llm.ctxs))
	}
	orgID, ok := llmdomain.OrgIDFromContext(llm.ctxs[0])
	if !ok || orgID != 1 {
		t.Fatalf("expected org 1 in completion context, got %d (ok=%v)", orgID, ok)
	}
	prompt := llm.calls[0].Prompt
	if !strings.Contains(prompt, "Problema con factura") || !strings.Contains(prompt, "No me llegó la factura de julio") {
		t.Fatalf("prompt missing ticket title/description: %q", prompt)
	}
}

func TestAITriageFencedJSON(t *testing.T) {
	llm := &recordingLLM{text: "```json\n{\"note\":\"nota con fences\",\"priority\":\"high\"}\n```"}
	svc, _ := newTriageHarness(t, llm, &triageMockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 50}})

	result, err := svc.Triage(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("triage failed: %v", err)
	}
	if result.Note != "nota con fences" {
		t.Fatalf("unexpected note: %q", result.Note)
	}
	if result.Priority == nil || *result.Priority != domain.PriorityHigh {
		t.Fatalf("expected high priority, got %v", result.Priority)
	}
}

func TestAITriageInvalidPriorityDropped(t *testing.T) {
	llm := &recordingLLM{text: `{"note":"Nota útil.","priority":"urgente"}`}
	svc, _ := newTriageHarness(t, llm, &triageMockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 50}})

	result, err := svc.Triage(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("triage failed: %v", err)
	}
	if result.Priority != nil {
		t.Fatalf("expected nil priority for invalid model value, got %v", result.Priority)
	}
	if result.Note != "Nota útil." {
		t.Fatalf("note must still be returned, got %q", result.Note)
	}
}

func TestAITriageMissingPriorityDropped(t *testing.T) {
	llm := &recordingLLM{text: `{"note":"Solo nota."}`}
	svc, _ := newTriageHarness(t, llm, &triageMockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 50}})

	result, err := svc.Triage(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("triage failed: %v", err)
	}
	if result.Priority != nil {
		t.Fatalf("expected nil priority when model omits it, got %v", result.Priority)
	}
}

func TestAITriageCreditsExhausted_NoLLMCall(t *testing.T) {
	llm := &recordingLLM{text: `{"note":"no debe usarse","priority":"alta"}`}
	svc, _ := newTriageHarness(t, llm, &triageMockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 0}})

	_, err := svc.Triage(context.Background(), 1, 1)
	if !errors.Is(err, ErrAiCreditsExhausted) {
		t.Fatalf("expected ErrAiCreditsExhausted, got %v", err)
	}
	if len(llm.calls) != 0 {
		t.Fatalf("exhausted credits must not call the LLM, got %d calls", len(llm.calls))
	}
}

func TestAITriageUnlimitedCreditsProceeds(t *testing.T) {
	llm := &recordingLLM{text: `{"note":"ok","priority":"media"}`}
	svc, _ := newTriageHarness(t, llm, &triageMockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 0, CreditsRemaining: 0}})

	result, err := svc.Triage(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("unlimited credits should not block: %v", err)
	}
	if result.Priority == nil || *result.Priority != domain.PriorityNormal {
		t.Fatalf("expected normal priority, got %v", result.Priority)
	}
}

func TestAITriageLedgerFailureFailsOpen(t *testing.T) {
	llm := &recordingLLM{text: `{"note":"Fail open.","priority":"baja"}`}
	svc, _ := newTriageHarness(t, llm, &triageMockBilling{err: errors.New("ledger down")})

	result, err := svc.Triage(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("ledger failure must fail open: %v", err)
	}
	if len(llm.calls) != 1 {
		t.Fatalf("expected LLM call despite ledger error, got %d", len(llm.calls))
	}
	if result.Priority == nil || *result.Priority != domain.PriorityLow {
		t.Fatalf("expected low priority, got %v", result.Priority)
	}
}

func TestAITriageMissingTicket_NotFound(t *testing.T) {
	llm := &recordingLLM{text: `{"note":"x","priority":"alta"}`}
	svc, _ := newTriageHarness(t, llm, &triageMockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 50}})

	_, err := svc.Triage(context.Background(), 1, 999)
	if !errors.Is(err, domain.ErrTicketNotFound) {
		t.Fatalf("expected ErrTicketNotFound for missing ticket, got %v", err)
	}
	if len(llm.calls) != 0 {
		t.Fatalf("missing ticket must not call the LLM, got %d calls", len(llm.calls))
	}
}

func TestAITriageForeignOrgTicket_NotFound(t *testing.T) {
	llm := &recordingLLM{text: `{"note":"x","priority":"alta"}`}
	svc, _ := newTriageHarness(t, llm, &triageMockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 50}})

	// Ticket 2 belongs to org 2; requesting it from org 1 is a 404-equivalent.
	_, err := svc.Triage(context.Background(), 1, 2)
	if !errors.Is(err, domain.ErrTicketNotFound) {
		t.Fatalf("expected ErrTicketNotFound for foreign-org ticket, got %v", err)
	}
	if len(llm.calls) != 0 {
		t.Fatalf("foreign-org ticket must not call the LLM, got %d calls", len(llm.calls))
	}
}

func TestAITriageLLMFailure(t *testing.T) {
	llm := &recordingLLM{err: errors.New("provider timeout")}
	svc, _ := newTriageHarness(t, llm, &triageMockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 50}})

	_, err := svc.Triage(context.Background(), 1, 1)
	if err == nil || !strings.Contains(err.Error(), "triage failed") {
		t.Fatalf("expected wrapped LLM error, got %v", err)
	}
}
