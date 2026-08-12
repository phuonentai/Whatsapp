package services

import (
	"context"
	"errors"
	"testing"

	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
)

func seedAnsweredRun(t *testing.T, runs *mockRunRepo, requiresHuman bool) (*domain.InquiryRun, int32, int32) {
	ctx := context.Background()
	run, _ := runs.CreateRun(ctx, 42, strPtr(""), "m")
	rec, _ := runs.CreateRecipient(ctx, 42, run.ID, 1, 101, strPtr("draft"))
	rec.Status = domain.RecipientAnswered
	run.Status = domain.RunAwaitingResponses
	resp := &domain.InquiryResponse{
		OrganizationID: 42, RecipientID: rec.ID, RawMessageID: "m1",
		Items:          []domain.ResponseItem{{ProductName: "A", Disponible: true, PrecioUnitario: f64(10000)}},
		Resumen:        "Disponible",
		RequiereHumano: requiresHuman,
	}
	_, _ = runs.SaveResponse(ctx, resp)
	return run, rec.ID, rec.SupplierID
}

func newOrderServiceDeps() (*mockRunRepo, *mockOrderRepo, *mockContacts, *mockAuditRepo, *CounterSink, *mockKillSwitch) {
	runs := newMockRunRepo()
	orders := newMockOrderRepo()
	contacts := newMockContacts()
	audit := &mockAuditRepo{}
	metrics := NewCounterSink()
	kill := &mockKillSwitch{}
	return runs, orders, contacts, audit, metrics, kill
}

func buildOrderService(runs *mockRunRepo, orders *mockOrderRepo, contacts *mockContacts, audit *mockAuditRepo, metrics *CounterSink, kill *mockKillSwitch) *orderService {
	return NewOrderService(runs, orders, contacts, audit, kill, metrics, stubLogger{})
}

func TestPlaceOrderRequiresAnsweredResponse(t *testing.T) {
	ctx := context.Background()
	runs, orders, contacts, audit, metrics, kill := newOrderServiceDeps()
	svc := buildOrderService(runs, orders, contacts, audit, metrics, kill)
	contacts.byID[101] = &domain.ContactRef{ID: 101, PhoneNumber: "+573001234567", ConsentStatus: "granted"}

	run, _ := runs.CreateRun(ctx, 42, strPtr(""), "m")
	run.Status = domain.RunAwaitingResponses
	_, _ = runs.CreateRecipient(ctx, 42, run.ID, 1, 101, strPtr("draft")) // no response

	_, err := svc.PlaceOrder(ctx, 42, "m1", PlaceOrderInput{RunID: run.ID, SupplierID: 1, Items: []domain.OrderItem{{ProductID: 10, Quantity: 2}}})
	if !errors.Is(err, domain.ErrResponseNotAnswered) {
		t.Fatalf("expected ErrResponseNotAnswered, got %v", err)
	}
	if len(orders.orders) != 0 {
		t.Fatalf("no order should be created")
	}
}

func TestPlaceOrderRequiresHumanNeedsOverride(t *testing.T) {
	ctx := context.Background()
	runs, orders, contacts, audit, metrics, kill := newOrderServiceDeps()
	svc := buildOrderService(runs, orders, contacts, audit, metrics, kill)
	contacts.byID[101] = &domain.ContactRef{ID: 101, PhoneNumber: "+573001234567", ConsentStatus: "granted"}

	run, recID, supID := seedAnsweredRun(t, runs, true)

	_, err := svc.PlaceOrder(ctx, 42, "m1", PlaceOrderInput{RunID: run.ID, SupplierID: supID, Items: []domain.OrderItem{{ProductID: 10, Quantity: 1}}})
	if !errors.Is(err, domain.ErrResponseRequiresHuman) {
		t.Fatalf("expected ErrResponseRequiresHuman, got %v", err)
	}

	// explicit org:manage override proceeds
	order, err := svc.PlaceOrder(ctx, 42, "m1", PlaceOrderInput{RunID: run.ID, SupplierID: supID, Items: []domain.OrderItem{{ProductID: 10, Quantity: 1}}, Override: true})
	if err != nil {
		t.Fatalf("override placement: %v", err)
	}
	if order.Status != domain.OrderPlaced {
		t.Fatalf("expected placed order, got %s", order.Status)
	}
	_ = recID
}

func TestPlaceOrderKillSwitchRecordsBlockedOrder(t *testing.T) {
	ctx := context.Background()
	runs, orders, contacts, audit, metrics, kill := newOrderServiceDeps()
	kill.on = true
	svc := buildOrderService(runs, orders, contacts, audit, metrics, kill)
	contacts.byID[101] = &domain.ContactRef{ID: 101, PhoneNumber: "+573001234567", ConsentStatus: "granted"}

	run, _, supID := seedAnsweredRun(t, runs, false)
	order, err := svc.PlaceOrder(ctx, 42, "m1", PlaceOrderInput{RunID: run.ID, SupplierID: supID, Items: []domain.OrderItem{{ProductID: 10, Quantity: 1}}})
	if err != nil {
		t.Fatalf("placement: %v", err)
	}
	if order.Status != domain.OrderSendBlocked || order.BlockedReason == nil || *order.BlockedReason != "kill_switch" {
		t.Fatalf("expected send_blocked(kill_switch), got %+v", order)
	}
	if metrics.Get(Key(MetricBlock, map[string]string{"org": "42", "reason": "kill_switch"})) != 1 {
		t.Fatalf("expected block metric")
	}
}

func TestPlaceOrderConsentWithdrawnBlocksSend(t *testing.T) {
	ctx := context.Background()
	runs, orders, contacts, audit, metrics, kill := newOrderServiceDeps()
	svc := buildOrderService(runs, orders, contacts, audit, metrics, kill)
	contacts.byID[101] = &domain.ContactRef{ID: 101, PhoneNumber: "+573001234567", ConsentStatus: "withdrawn"}

	run, _, supID := seedAnsweredRun(t, runs, false)
	order, err := svc.PlaceOrder(ctx, 42, "m1", PlaceOrderInput{RunID: run.ID, SupplierID: supID, Items: []domain.OrderItem{{ProductID: 10, Quantity: 1}}})
	if err != nil {
		t.Fatalf("placement: %v", err)
	}
	if order.Status != domain.OrderSendBlocked || *order.BlockedReason != "consent_withdrawn" {
		t.Fatalf("expected consent_withdrawn block, got %+v", order)
	}
}

func TestPlaceOrderRetryIsIdempotent(t *testing.T) {
	ctx := context.Background()
	runs, orders, contacts, audit, metrics, kill := newOrderServiceDeps()
	svc := buildOrderService(runs, orders, contacts, audit, metrics, kill)
	contacts.byID[101] = &domain.ContactRef{ID: 101, PhoneNumber: "+573001234567", ConsentStatus: "granted"}

	run, _, supID := seedAnsweredRun(t, runs, false)
	in := PlaceOrderInput{RunID: run.ID, SupplierID: supID, Items: []domain.OrderItem{{ProductID: 10, Quantity: 1}}}
	first, err := svc.PlaceOrder(ctx, 42, "m1", in)
	if err != nil {
		t.Fatalf("first placement: %v", err)
	}
	// retried POST (double-click / client retry)
	second, err := svc.PlaceOrder(ctx, 42, "m1", in)
	if err != nil {
		t.Fatalf("retried placement: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected the SAME order on retry (idempotent), got %d vs %d", second.ID, first.ID)
	}
	if len(orders.orders) != 1 {
		t.Fatalf("expected exactly 1 order row")
	}
}
