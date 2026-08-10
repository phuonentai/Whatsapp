package services

import (
	"context"
	"errors"
	"testing"

	"github.com/moasq/go-b2b-starter/internal/modules/campaigns/domain"
	billingServices "github.com/moasq/go-b2b-starter/internal/modules/billing/app/services"
	billingDomain "github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
	crmDomain "github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	llmdomain "github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
)

// ---------- mocks ----------

type mockLLM struct {
	llmdomain.LLMClient
	text string
	err  error
}

func (m *mockLLM) Complete(ctx context.Context, req llmdomain.CompletionRequest) (*llmdomain.CompletionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &llmdomain.CompletionResponse{Text: m.text, TokensUsed: 10}, nil
}

type mockBilling struct {
	billingServices.BillingService
	status *billingDomain.AiUsageStatus
	err    error
}

func (m *mockBilling) GetAiUsageStatus(ctx context.Context, orgID int32) (*billingDomain.AiUsageStatus, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.status, nil
}

type mockTagRepo struct {
	crmDomain.TagRepository
	tags []*crmDomain.Tag
}

func (m *mockTagRepo) List(ctx context.Context, orgID int32) ([]*crmDomain.Tag, error) {
	return m.tags, nil
}

// ---------- tests ----------

func TestAiBuildHappyPath(t *testing.T) {
	builder := NewAudienceBuilder(
		&mockLLM{text: `[{"field":"lead_status","op":"eq","value":"cliente"},{"field":"recency_days","op":"lte","value":30}]`},
		&mockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 50}},
		&mockTagRepo{},
		&mockEvaluator{count: &domain.EvalResult{Total: 25, ExcludedByGates: 5}},
	)

	result, err := builder.Build(context.Background(), 42, "clientes que escribieron este mes")
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if len(result.FilterSpec) != 2 {
		t.Fatalf("expected 2 filters, got %d", len(result.FilterSpec))
	}
	if result.Preview == nil || result.Preview.Total != 25 {
		t.Fatalf("unexpected preview: %+v", result.Preview)
	}
}

func TestAiBuildFencedJSON(t *testing.T) {
	builder := NewAudienceBuilder(
		&mockLLM{text: "```json\n[{\"field\":\"source\",\"op\":\"eq\",\"value\":\"whatsapp\"}]\n```"},
		&mockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 50}},
		&mockTagRepo{},
		&mockEvaluator{},
	)

	result, err := builder.Build(context.Background(), 42, "contactos de whatsapp")
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if result.FilterSpec[0].Field != "source" {
		t.Fatalf("unexpected spec: %+v", result.FilterSpec)
	}
}

func TestAiBuildInvalidOutputRejected(t *testing.T) {
	builder := NewAudienceBuilder(
		&mockLLM{text: `[{"field":"password","op":"eq","value":"x"}]`},
		&mockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 50}},
		&mockTagRepo{},
		&mockEvaluator{},
	)

	_, err := builder.Build(context.Background(), 42, "hack the plan")
	if !errors.Is(err, domain.ErrInvalidFilterSpec) {
		t.Fatalf("expected ErrInvalidFilterSpec, got %v", err)
	}
}

func TestAiBuildNonJSONOutputRejected(t *testing.T) {
	builder := NewAudienceBuilder(
		&mockLLM{text: "lo siento, no entendí"},
		&mockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 50}},
		&mockTagRepo{},
		&mockEvaluator{},
	)

	_, err := builder.Build(context.Background(), 42, "algo raro")
	if !errors.Is(err, domain.ErrInvalidFilterSpec) {
		t.Fatalf("expected ErrInvalidFilterSpec, got %v", err)
	}
}

func TestAiBuildCreditsExhaustedFailClosed(t *testing.T) {
	builder := NewAudienceBuilder(
		&mockLLM{text: `[{"field":"source","op":"eq","value":"whatsapp"}]`},
		&mockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 0}},
		&mockTagRepo{},
		&mockEvaluator{},
	)

	_, err := builder.Build(context.Background(), 42, "clientes")
	if !errors.Is(err, domain.ErrAiCreditsExhausted) {
		t.Fatalf("expected ErrAiCreditsExhausted, got %v", err)
	}
}

func TestAiBuildUnlimitedCreditsProceeds(t *testing.T) {
	builder := NewAudienceBuilder(
		&mockLLM{text: `[{"field":"source","op":"eq","value":"whatsapp"}]`},
		&mockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 0, CreditsRemaining: 0}},
		&mockTagRepo{},
		&mockEvaluator{},
	)

	if _, err := builder.Build(context.Background(), 42, "clientes"); err != nil {
		t.Fatalf("unlimited credits should not block: %v", err)
	}
}

func TestAiBuildTagIDMustBelongToOrg(t *testing.T) {
	builder := NewAudienceBuilder(
		&mockLLM{text: `[{"field":"tag_ids","op":"any","value":[99]}]`},
		&mockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 50}},
		&mockTagRepo{tags: []*crmDomain.Tag{{ID: 1, Nombre: "mayorista"}}},
		&mockEvaluator{},
	)

	_, err := builder.Build(context.Background(), 42, "mayoristas")
	if !errors.Is(err, domain.ErrInvalidFilterSpec) {
		t.Fatalf("expected ErrInvalidFilterSpec for foreign tag id, got %v", err)
	}
}

func TestAiBuildEmptyDescription(t *testing.T) {
	builder := NewAudienceBuilder(&mockLLM{}, &mockBilling{}, &mockTagRepo{}, &mockEvaluator{})
	_, err := builder.Build(context.Background(), 42, "  ")
	if !errors.Is(err, domain.ErrInvalidFilterSpec) {
		t.Fatalf("expected ErrInvalidFilterSpec, got %v", err)
	}
}
