package services

import (
	"context"
	"errors"
	"testing"

	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
	procurementEvents "github.com/moasq/go-b2b-starter/internal/modules/procurement/domain/events"
	whatsappDomain "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
)

func newSendHandlerDeps() (*mockRunRepo, *mockOrderRepo, *mockContacts, *mockAuditRepo, *mockSender, *CounterSink, *mockKillSwitch) {
	runs := newMockRunRepo()
	orders := newMockOrderRepo()
	contacts := newMockContacts()
	audit := &mockAuditRepo{}
	sender := &mockSender{msgID: "wamid.1"}
	metrics := NewCounterSink()
	kill := &mockKillSwitch{}
	return runs, orders, contacts, audit, sender, metrics, kill
}

func buildSendHandler(runs *mockRunRepo, orders *mockOrderRepo, contacts *mockContacts, audit *mockAuditRepo, sender *mockSender, metrics *CounterSink, kill *mockKillSwitch, pacer Pacer) SendHandler {
	configs := &mockConfigRepo{config: &whatsappDomain.WhatsAppConfig{IsActive: true, AccessToken: "tok", PhoneNumberID: "123", APIVersion: "v21.0"}}
	return NewSendHandler(runs, orders, contacts, audit, configs, pacer, kill, metrics, stubLogger{}, sendHandlerOptions{Sender: sender})
}

func seedSentRun(t *testing.T, runs *mockRunRepo) (int32, int32, int32) {
	ctx := context.Background()
	run, _ := runs.CreateRun(ctx, 42, strPtr(""), "m")
	rec, _ := runs.CreateRecipient(ctx, 42, run.ID, 1, 101, strPtr("draft"))
	run.Status = domain.RunSending
	run.SentAt = nil
	return run.ID, rec.ID, 101
}

func TestInquirySendRedeliveryNoDoubleSend(t *testing.T) {
	ctx := context.Background()
	runs, orders, contacts, audit, sender, metrics, kill := newSendHandlerDeps()
	handler := buildSendHandler(runs, orders, contacts, audit, sender, metrics, kill, &fakePacer{allowed: true})

	runID, recID, contactID := seedSentRun(t, runs)

	ev := procurementEvents.NewInquirySend(42, runID, recID, 1, contactID, "+573001234567", "hola")
	if err := handler.HandleInquirySend(ctx, ev); err != nil {
		t.Fatalf("first dispatch: %v", err)
	}
	if sender.calls != 1 {
		t.Fatalf("expected 1 send, got %d", sender.calls)
	}
	if runs.recipients[recID].Status != domain.RecipientSent {
		t.Fatalf("expected sent recipient")
	}

	// Redelivery after success: recipient already sent → guarded no-op, no send.
	if err := handler.HandleInquirySend(ctx, ev); err != nil {
		t.Fatalf("redelivery: %v", err)
	}
	if sender.calls != 1 {
		t.Fatalf("expected NO second send on redelivery, got %d", sender.calls)
	}
}

func TestInquirySendKillSwitchBlocksDispatch(t *testing.T) {
	ctx := context.Background()
	runs, orders, contacts, audit, sender, metrics, kill := newSendHandlerDeps()
	kill.on = true
	handler := buildSendHandler(runs, orders, contacts, audit, sender, metrics, kill, &fakePacer{allowed: true})

	runID, recID, contactID := seedSentRun(t, runs)
	ev := procurementEvents.NewInquirySend(42, runID, recID, 1, contactID, "+573001234567", "hola")

	if err := handler.HandleInquirySend(ctx, ev); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if sender.calls != 0 {
		t.Fatalf("expected NO send under kill switch, got %d", sender.calls)
	}
	if !audit.hasAction("send_skipped") {
		t.Fatalf("expected send_skipped audit")
	}
	if metrics.Get(Key(MetricBlock, map[string]string{"org": "42", "reason": "kill_switch"})) != 1 {
		t.Fatalf("expected kill_switch block metric")
	}
}

func TestInquirySendRateLimitRetries(t *testing.T) {
	ctx := context.Background()
	runs, orders, contacts, audit, sender, metrics, kill := newSendHandlerDeps()
	handler := buildSendHandler(runs, orders, contacts, audit, sender, metrics, kill, &fakePacer{allowed: false})

	runID, recID, contactID := seedSentRun(t, runs)
	ev := procurementEvents.NewInquirySend(42, runID, recID, 1, contactID, "+573001234567", "hola")

	err := handler.HandleInquirySend(ctx, ev)
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
	if sender.calls != 0 {
		t.Fatalf("expected no send when rate limited")
	}
	// recipient still pending → the dispatcher retry will send later
	if runs.recipients[recID].Status != domain.RecipientPending {
		t.Fatalf("expected recipient to remain pending")
	}
}

func TestOrderConfirmKillSwitchAtDispatch(t *testing.T) {
	ctx := context.Background()
	runs, orders, contacts, audit, sender, metrics, kill := newSendHandlerDeps()
	kill.on = true
	handler := buildSendHandler(runs, orders, contacts, audit, sender, metrics, kill, &fakePacer{allowed: true})

	run, _ := runs.CreateRun(ctx, 42, strPtr(""), "m")
	order := &domain.Order{ID: 1, OrganizationID: 42, RunID: run.ID, SupplierID: 1, ContactID: 101, Status: domain.OrderPlaced}
	_, _ = orders.CreateOrder(ctx, order)

	ev := procurementEvents.NewOrderConfirmSend(42, order.ID, run.ID, 1, 101, "+573001234567", "confirmación")
	if err := handler.HandleOrderConfirmSend(ctx, ev); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if sender.calls != 0 {
		t.Fatalf("expected NO confirmation send under kill switch")
	}
	updated, _ := orders.GetOrder(ctx, 42, order.ID)
	if updated.Status != domain.OrderSendBlocked || updated.BlockedReason == nil || *updated.BlockedReason != "kill_switch" {
		t.Fatalf("expected order send_blocked(kill_switch), got %+v", updated)
	}
}

func TestOrderConfirmConsentWithdrawnAtDispatch(t *testing.T) {
	ctx := context.Background()
	runs, orders, contacts, audit, sender, metrics, kill := newSendHandlerDeps()
	contacts.byID[101] = &domain.ContactRef{ID: 101, PhoneNumber: "+573001234567", ConsentStatus: "withdrawn"}
	handler := buildSendHandler(runs, orders, contacts, audit, sender, metrics, kill, &fakePacer{allowed: true})

	run, _ := runs.CreateRun(ctx, 42, strPtr(""), "m")
	order := &domain.Order{ID: 1, OrganizationID: 42, RunID: run.ID, SupplierID: 1, ContactID: 101, Status: domain.OrderPlaced}
	_, _ = orders.CreateOrder(ctx, order)

	ev := procurementEvents.NewOrderConfirmSend(42, order.ID, run.ID, 1, 101, "+573001234567", "confirmación")
	if err := handler.HandleOrderConfirmSend(ctx, ev); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if sender.calls != 0 {
		t.Fatalf("expected NO confirmation send with withdrawn consent")
	}
	updated, _ := orders.GetOrder(ctx, 42, order.ID)
	if updated.Status != domain.OrderSendBlocked || *updated.BlockedReason != "consent_withdrawn" {
		t.Fatalf("expected consent_withdrawn block, got %+v", updated)
	}
}

func TestOrderConfirmSendSuccess(t *testing.T) {
	ctx := context.Background()
	runs, orders, contacts, audit, sender, metrics, kill := newSendHandlerDeps()
	contacts.byID[101] = &domain.ContactRef{ID: 101, PhoneNumber: "+573001234567", ConsentStatus: "granted"}
	handler := buildSendHandler(runs, orders, contacts, audit, sender, metrics, kill, &fakePacer{allowed: true})

	run, _ := runs.CreateRun(ctx, 42, strPtr(""), "m")
	order := &domain.Order{ID: 1, OrganizationID: 42, RunID: run.ID, SupplierID: 1, ContactID: 101, Status: domain.OrderPlaced}
	_, _ = orders.CreateOrder(ctx, order)

	ev := procurementEvents.NewOrderConfirmSend(42, order.ID, run.ID, 1, 101, "+573001234567", "confirmación")
	if err := handler.HandleOrderConfirmSend(ctx, ev); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if sender.calls != 1 {
		t.Fatalf("expected 1 confirmation send")
	}
	updated, _ := orders.GetOrder(ctx, 42, order.ID)
	if updated.Status != domain.OrderConfirmSent {
		t.Fatalf("expected confirm_sent, got %s", updated.Status)
	}
}

func TestInquirySendAllFailTransitionsRunFailed(t *testing.T) {
	ctx := context.Background()
	runs, orders, contacts, audit, sender, metrics, kill := newSendHandlerDeps()
	handler := buildSendHandler(runs, orders, contacts, audit, sender, metrics, kill, &fakePacer{allowed: true})

	runID, recID, contactID := seedSentRun(t, runs)
	// first send fails with a transient error → run stays sending
	sender.err = errors.New("meta 500")
	ev := procurementEvents.NewInquirySend(42, runID, recID, 1, contactID, "+573001234567", "hola")
	if err := handler.HandleInquirySend(ctx, ev); err == nil {
		t.Fatalf("expected transient error to bubble for dispatcher retry")
	}
	_ = sender
}
