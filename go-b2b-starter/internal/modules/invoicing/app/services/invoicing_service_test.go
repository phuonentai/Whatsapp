package services

import (
	"context"
	"testing"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	crmEvents "github.com/moasq/go-b2b-starter/internal/modules/crm/domain/events"
	invdomain "github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
)

func dealWithContact() *domain.DealWithRefs {
	contactID := int32(11)
	companyID := int32(5)
	monto := 150.0
	return &domain.DealWithRefs{
		Deal: domain.Deal{
			ID: 99, OrganizationID: 7, Nombre: "Negocio test",
			ContactID: &contactID, CompanyID: &companyID, Monto: &monto, Moneda: "COP",
		},
	}
}

func TestCreateForDeal_CreatesInvoiceOnce(t *testing.T) {
	repo := newMockInvoiceRepo()
	provider := &mockProvider{
		customer: &invdomain.CustomerRef{ExternalID: "cust-1"},
		created: &invdomain.Invoice{
			OrganizationID: 7, DealID: 99, ExternalID: "inv-1", Status: invdomain.InvoiceStatusPending,
		},
	}
	svc, _ := newTestService(repo, provider)

	svc.dealRepo = &mockDealRepo{deals: map[int32]*domain.DealWithRefs{99: dealWithContact()}}
	svc.companyRepo = &mockCompanyRepo{companies: map[int32]*domain.CompanyWithCounts{5: {Company: domain.Company{ID: 5, Name: "Acme SAS", Nit: "900123456"}}}}
	svc.contactRepo = &mockContactRepo{contacts: map[int32]*domain.Contact{11: {ID: 11, PhoneNumber: "+573001234567", DisplayName: "Juan"}}}
	svc.convRepo = &mockConvRepo{byContact: map[int32]*domain.Conversation{11: {ID: 3, ContactID: 11}}}

	first, err := svc.CreateForDeal(context.Background(), 7, 99)
	if err != nil {
		t.Fatalf("first CreateForDeal failed: %v", err)
	}
	if first.ExternalID != "inv-1" {
		t.Fatalf("unexpected invoice %+v", first)
	}

	// Duplicate trigger must NOT create a second provider invoice.
	second, err := svc.CreateForDeal(context.Background(), 7, 99)
	if err != nil {
		t.Fatalf("second CreateForDeal failed: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected same invoice on re-trigger, got different IDs")
	}
	if provider.createCalls != 1 {
		t.Fatalf("expected exactly 1 provider invoice call, got %d", provider.createCalls)
	}
}

func TestProcessWebhookEvent_UpdatesAndIsIdempotent(t *testing.T) {
	repo := newMockInvoiceRepo()
	_, _ = repo.Insert(context.Background(), &invdomain.Invoice{
		OrganizationID: 7, DealID: 99, ExternalID: "inv-1", Status: invdomain.InvoiceStatusPending,
	})
	provider := &mockProvider{}
	svc, _ := newTestService(repo, provider)
	svc.dealRepo = &mockDealRepo{deals: map[int32]*domain.DealWithRefs{99: dealWithContact()}}
	svc.contactRepo = &mockContactRepo{contacts: map[int32]*domain.Contact{11: {ID: 11, PhoneNumber: "+573001234567"}}}
	svc.convRepo = &mockConvRepo{byContact: map[int32]*domain.Conversation{11: {ID: 3, ContactID: 11}}}

	body := `{"id":"inv-1","status":"valid","cufe":"CUFE","pdf_url":"https://pdf"}`
	if err := svc.ProcessWebhookEvent(context.Background(), []byte(body)); err != nil {
		t.Fatalf("ProcessWebhookEvent failed: %v", err)
	}
	updated, _ := repo.GetByDeal(context.Background(), 7, 99)
	if updated.Status != invdomain.InvoiceStatusValid || updated.Cufe != "CUFE" {
		t.Fatalf("unexpected updated invoice %+v", updated)
	}

	// Duplicate event with same status: no-op.
	if err := svc.ProcessWebhookEvent(context.Background(), []byte(body)); err != nil {
		t.Fatalf("duplicate webhook failed: %v", err)
	}

	// Regression to pending must be ignored (invoice already final).
	regress := `{"id":"inv-1","status":"pending"}`
	if err := svc.ProcessWebhookEvent(context.Background(), []byte(regress)); err != nil {
		t.Fatalf("regression webhook failed: %v", err)
	}
	final, _ := repo.GetByDeal(context.Background(), 7, 99)
	if final.Status != invdomain.InvoiceStatusValid {
		t.Fatalf("status regressed to %s", final.Status)
	}
}

func TestProcessWebhookEvent_BadSignaturePayload(t *testing.T) {
	repo := newMockInvoiceRepo()
	svc, _ := newTestService(repo, &mockProvider{})

	if err := svc.ProcessWebhookEvent(context.Background(), []byte("not-json")); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if err := svc.ProcessWebhookEvent(context.Background(), []byte(`{"id":"nope","status":"valid"}`)); err != nil {
		t.Fatalf("unknown external id should be a logged no-op, got %v", err)
	}
}

func TestPollPending_ReconcilesAndNotifiesOnce(t *testing.T) {
	repo := newMockInvoiceRepo()
	inv, _ := repo.Insert(context.Background(), &invdomain.Invoice{
		OrganizationID: 7, DealID: 99, ExternalID: "inv-1", Status: invdomain.InvoiceStatusPending,
	})
	provider := &mockProvider{
		status: &invdomain.Invoice{ExternalID: "inv-1", Status: invdomain.InvoiceStatusValid, Cufe: "C", PdfURL: "p"},
	}
	svc, out := newTestService(repo, provider)
	svc.dealRepo = &mockDealRepo{deals: map[int32]*domain.DealWithRefs{99: dealWithContact()}}
	svc.contactRepo = &mockContactRepo{contacts: map[int32]*domain.Contact{11: {ID: 11, PhoneNumber: "+573001234567"}}}
	svc.convRepo = &mockConvRepo{byContact: map[int32]*domain.Conversation{11: {ID: 3, ContactID: 11}}}
	repo.pending = []*invdomain.Invoice{inv}

	reconciled, err := svc.PollPending(context.Background())
	if err != nil {
		t.Fatalf("PollPending failed: %v", err)
	}
	if reconciled != 1 {
		t.Fatalf("expected 1 reconciled, got %d", reconciled)
	}
	if len(out.sent) != 1 {
		t.Fatalf("expected 1 notification, got %d: %v", len(out.sent), out.sent)
	}

	// Second poll: invoice now valid, remote unchanged → no re-notify.
	provider.status = &invdomain.Invoice{ExternalID: "inv-1", Status: invdomain.InvoiceStatusValid}
	reconciled, err = svc.PollPending(context.Background())
	if err != nil {
		t.Fatalf("second PollPending failed: %v", err)
	}
	if len(out.sent) != 1 {
		t.Fatalf("expected no new notification on second poll, got %d", len(out.sent))
	}
}

func TestDealStageListener_OnlyTriggersOnFacturado(t *testing.T) {
	repo := newMockInvoiceRepo()
	provider := &mockProvider{
		customer: &invdomain.CustomerRef{ExternalID: "cust-1"},
		created: &invdomain.Invoice{
			OrganizationID: 7, DealID: 99, ExternalID: "inv-1", Status: invdomain.InvoiceStatusPending,
		},
	}
	svc, _ := newTestService(repo, provider)
	svc.dealRepo = &mockDealRepo{deals: map[int32]*domain.DealWithRefs{99: dealWithContact()}}
	svc.companyRepo = &mockCompanyRepo{companies: map[int32]*domain.CompanyWithCounts{5: {Company: domain.Company{ID: 5, Name: "Acme", Nit: "900123"}}}}
	svc.contactRepo = &mockContactRepo{contacts: map[int32]*domain.Contact{11: {ID: 11, PhoneNumber: "+573001234567"}}}
	svc.convRepo = &mockConvRepo{byContact: map[int32]*domain.Conversation{11: {ID: 3, ContactID: 11}}}

	listener := NewDealStageListener(svc, nopLogger{})
	evt := func(stage string) *crmEvents.DealStageChanged {
		return &crmEvents.DealStageChanged{OrganizationID: 7, DealID: 99, NewStageName: stage}
	}

	if err := listener.HandleStageChanged(context.Background(), evt("cotizacion")); err != nil {
		t.Fatalf("non-facturado stage should not error: %v", err)
	}
	if provider.createCalls != 0 {
		t.Fatalf("expected no invoice for non-facturado stage, got %d", provider.createCalls)
	}

	if err := listener.HandleStageChanged(context.Background(), evt("facturado")); err != nil {
		t.Fatalf("facturado stage failed: %v", err)
	}
	if provider.createCalls != 1 {
		t.Fatalf("expected 1 invoice for facturado, got %d", provider.createCalls)
	}
}
