package services

import (
	"context"
	"testing"

	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
	whatsappEvents "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
)

func newSubscriberDeps() (*mockRunRepo, *mockAuditRepo, *mockLLM, *CounterSink) {
	runs := newMockRunRepo()
	audit := &mockAuditRepo{}
	llm := &mockLLM{}
	metrics := NewCounterSink()
	return runs, audit, llm, metrics
}

func buildSubscriber(runs *mockRunRepo, audit *mockAuditRepo, llm *mockLLM, metrics *CounterSink) *ProcurementSubscriber {
	return NewProcurementSubscriber(runs, audit, NewExtractionService(llm, &mockBilling{}, metrics, stubLogger{}), metrics, stubLogger{})
}

func eventFrom(phone, content string) *whatsappEvents.MessageReceived {
	return &whatsappEvents.MessageReceived{
		OrganizationID: 42,
		From:           phone,
		MessageType:    "text",
		Content:        content,
		MessageSID:     "wamid.inbound.1",
	}
}

func TestSubscriberNonRecipientNoLLMCall(t *testing.T) {
	ctx := context.Background()
	runs, audit, llm, metrics := newSubscriberDeps()
	sub := buildSubscriber(runs, audit, llm, metrics)

	if err := sub.HandleMessageReceived(ctx, eventFrom("+573009999999", "hola")); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if llm.calls != 0 {
		t.Fatalf("expected NO LLM call for non-recipient message")
	}
}

func TestSubscriberExtractsExactlyOnce(t *testing.T) {
	ctx := context.Background()
	runs, audit, llm, metrics := newSubscriberDeps()
	run, _ := runs.CreateRun(ctx, 42, strPtr(""), "m")
	rec, _ := runs.CreateRecipient(ctx, 42, run.ID, 1, 101, strPtr("draft"))
	run.Status = domain.RunAwaitingResponses
	rec.Status = domain.RecipientSent
	now := nowTime()
	rec.SentAt = &now
	runs.byPhone["+573001234567"] = []*domain.InquiryRecipient{rec}
	llm.addResponse(`{"items":[{"product_name":"Papel","disponible":true,"precio_unitario":10000}],"resumen":"Disponible","requiere_humano":false}`)

	sub := buildSubscriber(runs, audit, llm, metrics)
	if err := sub.HandleMessageReceived(ctx, eventFrom("+573001234567", "Disponible a 10mil")); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if llm.calls != 1 {
		t.Fatalf("expected exactly 1 metered extraction call, got %d", llm.calls)
	}
	if rec.Status != domain.RecipientAnswered {
		t.Fatalf("expected answered recipient, got %s", rec.Status)
	}
	// run completed: 1 recipient answered of 1
	if runs.runs[run.ID].Status != domain.RunCompleted {
		t.Fatalf("expected completed run, got %s", runs.runs[run.ID].Status)
	}

	// Redelivery: same raw message id → no second extraction, no second row.
	if err := sub.HandleMessageReceived(ctx, eventFrom("+573001234567", "Disponible a 10mil")); err != nil {
		t.Fatalf("redelivery: %v", err)
	}
	if llm.calls != 1 {
		t.Fatalf("expected NO second extraction on redelivery, got %d", llm.calls)
	}
}

func TestSubscriberRequiereHumanoEscalates(t *testing.T) {
	ctx := context.Background()
	runs, audit, llm, metrics := newSubscriberDeps()
	run, _ := runs.CreateRun(ctx, 42, strPtr(""), "m")
	rec, _ := runs.CreateRecipient(ctx, 42, run.ID, 1, 101, strPtr("draft"))
	run.Status = domain.RunAwaitingResponses
	rec.Status = domain.RecipientSent
	runs.byPhone["+573001234567"] = []*domain.InquiryRecipient{rec}
	llm.addResponse(`{"items":[],"resumen":"Negocia precio","requiere_humano":true}`)

	sub := buildSubscriber(runs, audit, llm, metrics)
	if err := sub.HandleMessageReceived(ctx, eventFrom("+573001234567", "negociamos el precio")); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if runs.runs[run.ID].Status != domain.RunEscalated {
		t.Fatalf("expected escalated run (no auto-quote), got %s", runs.runs[run.ID].Status)
	}
	if metrics.Get(Key(MetricExtractionEscalated, map[string]string{"org": "42"})) != 1 {
		t.Fatalf("expected extraction escalation metric")
	}
	if !audit.hasAction("escalate") {
		t.Fatalf("expected escalate audit")
	}
}

func TestSubscriberCreditsExhaustedEscalatesWithoutUnmeteredCall(t *testing.T) {
	ctx := context.Background()
	runs, audit, llm, metrics := newSubscriberDeps()
	run, _ := runs.CreateRun(ctx, 42, strPtr(""), "m")
	rec, _ := runs.CreateRecipient(ctx, 42, run.ID, 1, 101, strPtr("draft"))
	run.Status = domain.RunSending
	rec.Status = domain.RecipientSent
	runs.byPhone["+573001234567"] = []*domain.InquiryRecipient{rec}

	sub := NewProcurementSubscriber(runs, audit, NewExtractionService(llm, &mockBilling{exhausted: true}, metrics, stubLogger{}), metrics, stubLogger{})
	if err := sub.HandleMessageReceived(ctx, eventFrom("+573001234567", "disponible")); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if llm.calls != 0 {
		t.Fatalf("expected ZERO unmetered extraction calls, got %d", llm.calls)
	}
	if runs.runs[run.ID].Status != domain.RunEscalated {
		t.Fatalf("expected escalated run on credit exhaustion")
	}
}
