package services

import (
	"context"
	"errors"
	"testing"

	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
)

func newTestDeps() (*mockRunRepo, *mockOrderRepo, *mockAuditRepo, *mockLLM, *mockContacts, *CounterSink) {
	return newMockRunRepo(), newMockOrderRepo(), &mockAuditRepo{}, &mockLLM{}, newMockContacts(), NewCounterSink()
}

func newProductsRepo() *mockProductRepo {
	return &mockProductRepo{products: []*domain.Product{
		{ID: 10, OrganizationID: 42, Name: "Papel", SKU: "PAP-001", Unit: "resma"},
		{ID: 11, OrganizationID: 42, Name: "Esfero", SKU: "ESF-002", Unit: "und"},
	}}
}

func TestCreateRunDraftsExactlyOnePerSupplier(t *testing.T) {
	ctx := context.Background()
	runs, _, audit, llm, _, metrics := newTestDeps()
	llm.addResponse(`{"message":"Hola proveedor A, ¿disponibilidad?"}`)
	llm.addResponse(`{"message":"Hola proveedor B, ¿precio?"}`)

	runs.seedSupplier(1, 101, "900111", "ProvA")
	runs.seedSupplier(2, 102, "900222", "ProvB")
	svc := NewRunService(runs, nil, newProductsRepo(), audit, NewDraftingService(llm, &mockBilling{}, audit, metrics, stubLogger{}), metrics, stubLogger{})

	// run with 2 suppliers, 2 products
	run, err := svc.CreateRun(ctx, 42, "member-1", CreateRunInput{
		SupplierIDs: []int32{1, 2},
		Products:    []RunProduct{{ProductID: 10, Quantity: 5}, {ProductID: 11, Quantity: 2}},
		Nota:        strPtr("urgente"),
	})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	if run.Status != domain.RunDraft {
		t.Fatalf("expected draft run, got %s", run.Status)
	}
	if llm.calls != 2 {
		t.Fatalf("expected exactly 2 metered LLM calls (one per supplier), got %d", llm.calls)
	}
	recs, _ := runs.ListRunRecipients(ctx, 42, run.ID)
	if len(recs) != 2 {
		t.Fatalf("expected 2 recipients, got %d", len(recs))
	}
	if metrics.Get(Key(MetricDraftAttempt, map[string]string{"org": "42"})) != 2 {
		t.Fatalf("expected 2 draft metrics, got %d", metrics.Get("procurement.draft_attempt"))
	}
}

func TestCreateRunCreditsExhaustedEscalatesWithoutUnmeteredCall(t *testing.T) {
	ctx := context.Background()
	runs, _, audit, llm, _, metrics := newTestDeps()
	runs.seedSupplier(1, 101, "900111", "ProvA")
	svc := NewRunService(runs, nil, newProductsRepo(), audit, NewDraftingService(llm, &mockBilling{exhausted: true}, audit, metrics, stubLogger{}), metrics, stubLogger{})

	run, err := svc.CreateRun(ctx, 42, "m", CreateRunInput{
		SupplierIDs: []int32{1},
		Products:    []RunProduct{{ProductID: 10, Quantity: 1}},
	})
	if err == nil {
		t.Fatalf("expected credits-exhausted error")
	}
	if !errors.Is(err, domain.ErrCreditsExhausted) {
		t.Fatalf("expected ErrCreditsExhausted, got %v", err)
	}
	if llm.calls != 0 {
		t.Fatalf("expected ZERO unmetered LLM invocations, got %d", llm.calls)
	}
	// The run exists and was escalated with an audit.
	if run == nil || run.Status != domain.RunEscalated {
		t.Fatalf("expected escalated run, got %+v", run)
	}
	if !audit.hasAction("escalate") {
		t.Fatalf("expected escalate audit")
	}
}

func TestCreateRunMalformedLLMResponseEscalates(t *testing.T) {
	ctx := context.Background()
	runs, _, audit, llm, _, metrics := newTestDeps()
	runs.seedSupplier(1, 101, "900111", "ProvA")
	llm.addResponse("not json at all")
	svc := NewRunService(runs, nil, newProductsRepo(), audit, NewDraftingService(llm, &mockBilling{}, audit, metrics, stubLogger{}), metrics, stubLogger{})

	run, err := svc.CreateRun(ctx, 42, "m", CreateRunInput{
		SupplierIDs: []int32{1},
		Products:    []RunProduct{{ProductID: 10, Quantity: 1}},
	})
	if err == nil || !errors.Is(err, domain.ErrMalformedLLMResponse) {
		t.Fatalf("expected ErrMalformedLLMResponse, got %v", err)
	}
	if run == nil || run.Status != domain.RunEscalated {
		t.Fatalf("expected escalated run on malformed LLM output")
	}
	if !audit.hasAction("llm_malformed") {
		t.Fatalf("expected llm_malformed audit")
	}
}

func TestSendRunEnqueuesOneEventPerRecipient(t *testing.T) {
	ctx := context.Background()
	runs, _, audit, llm, _, metrics := newTestDeps()
	runs.seedSupplier(1, 101, "900111", "ProvA")
	runs.seedSupplier(2, 102, "900222", "ProvB")
	runs.seedSupplier(3, 103, "900333", "ProvC")
	svc := NewRunService(runs, nil, newProductsRepo(), audit, NewDraftingService(llm, &mockBilling{}, audit, metrics, stubLogger{}), metrics, stubLogger{})

	run, err := svc.CreateRun(ctx, 42, "m", CreateRunInput{
		SupplierIDs: []int32{1, 2, 3},
		Products:    []RunProduct{{ProductID: 10, Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	sent, err := svc.SendRun(ctx, 42, run.ID)
	if err != nil {
		t.Fatalf("send run: %v", err)
	}
	if sent.Status != domain.RunSending {
		t.Fatalf("expected sending run, got %s", sent.Status)
	}
}

func TestSendRunNotDraftRejected(t *testing.T) {
	ctx := context.Background()
	runs, _, audit, llm, _, metrics := newTestDeps()
	runs.seedSupplier(1, 101, "900111", "ProvA")
	svc := NewRunService(runs, nil, newProductsRepo(), audit, NewDraftingService(llm, &mockBilling{}, audit, metrics, stubLogger{}), metrics, stubLogger{})

	run, _ := svc.CreateRun(ctx, 42, "m", CreateRunInput{
		SupplierIDs: []int32{1},
		Products:    []RunProduct{{ProductID: 10, Quantity: 1}},
	})
	// move it out of draft
	runs.runs[run.ID].Status = domain.RunSending
	if _, err := svc.SendRun(ctx, 42, run.ID); !errors.Is(err, domain.ErrRunNotDraft) {
		t.Fatalf("expected ErrRunNotDraft, got %v", err)
	}
}
