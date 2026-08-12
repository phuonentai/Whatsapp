package services

import (
	"context"
	"strings"
	"testing"

	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
	llmdomain "github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
)

func TestDraftingPromptUsesDisplayNameNotPII(t *testing.T) {
	ctx := context.Background()
	llm := &mockLLM{}
	audit := &mockAuditRepo{}
	metrics := NewCounterSink()
	svc := NewDraftingService(llm, &mockBilling{}, audit, metrics, stubLogger{})

	supplier := &domain.Supplier{ID: 1, NIT: "900111222", ContactID: 5}
	products := []*domain.Product{
		{ID: 10, Name: "Papel carta", SKU: "PAP-001", Unit: "resma"},
		{ID: 11, Name: "Esfero negro", SKU: "ESF-002", Unit: "und"},
	}
	qty := map[int32]float64{10: 5, 11: 24}

	msg, err := svc.DraftForSupplier(ctx, 42, supplier, "Proveedor Andino S.A.S.", products, qty)
	if err != nil {
		t.Fatalf("draft: %v", err)
	}
	if msg == "" {
		t.Fatalf("expected drafted message")
	}

	prompt := llm.prompts[0]
	if !strings.Contains(prompt, "Proveedor Andino S.A.S.") {
		t.Fatalf("expected display name (business identity allowlist) in prompt")
	}
	// The NIT/document number and phone must NOT appear.
	if strings.Contains(prompt, "900111222") {
		t.Fatalf("NIT must not appear in the prompt")
	}
	if strings.Contains(prompt, "+57") {
		t.Fatalf("phone must not appear in the prompt")
	}
	if !strings.Contains(prompt, "Papel carta") || !strings.Contains(prompt, "5 resma") {
		t.Fatalf("expected product name × quantity in prompt")
	}
}

func TestDraftingMeteredOrgContext(t *testing.T) {
	ctx := context.Background()
	llm := &ctxCapturingLLM{}
	audit := &mockAuditRepo{}
	metrics := NewCounterSink()
	svc := NewDraftingService(llm, &mockBilling{}, audit, metrics, stubLogger{})

	if _, err := svc.DraftForSupplier(ctx, 77, &domain.Supplier{ID: 1, NIT: "x"}, "Prov", nil, nil); err != nil {
		t.Fatalf("draft: %v", err)
	}
	orgID, ok := llmdomain.OrgIDFromContext(llm.lastCtx)
	if !ok || orgID != 77 {
		t.Fatalf("expected org id 77 in LLM context (metered ledger recording), got %d ok=%v", orgID, ok)
	}
}

func TestParseDraftMessageToleratesFencedJSON(t *testing.T) {
	msg, err := parseDraftMessage("```json\n{\"message\": \"Hola proveedor\"}\n```")
	if err != nil {
		t.Fatalf("parse fenced: %v", err)
	}
	if msg != "Hola proveedor" {
		t.Fatalf("expected message, got %q", msg)
	}
	if _, err := parseDraftMessage("nope"); err == nil {
		t.Fatalf("expected malformed error")
	}
}

func TestExtractionParsesContract(t *testing.T) {
	out, err := parseExtractionResult(`{"items":[{"product_name":"Papel","disponible":true,"precio_unitario":12000}],"resumen":"Disponible a 12k","requiere_humano":false}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(out.Items) != 1 || !out.Items[0].Disponible {
		t.Fatalf("unexpected items: %+v", out.Items)
	}
	if out.RequiereHumano {
		t.Fatalf("expected no human flag")
	}
}
