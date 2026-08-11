package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	invdomain "github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
)

// fakeLinker records the arguments and returns the configured outcome.
type fakeLinker struct {
	link string
	err  error
	calls int
	orgID, dealID int32
	amountCOP int64
}

func (f *fakeLinker) PaymentLink(ctx context.Context, orgID, dealID int32, amountCOP int64) (string, error) {
	f.calls++
	f.orgID, f.dealID, f.amountCOP = orgID, dealID, amountCOP
	return f.link, f.err
}

func setupNotifyEnv(svc *invoicingService) {
	svc.dealRepo = &mockDealRepo{deals: map[int32]*domain.DealWithRefs{99: dealWithContact()}}
	svc.companyRepo = &mockCompanyRepo{companies: map[int32]*domain.CompanyWithCounts{5: {Company: domain.Company{ID: 5, Name: "Acme SAS", Nit: "900123456"}}}}
	svc.contactRepo = &mockContactRepo{contacts: map[int32]*domain.Contact{11: {ID: 11, PhoneNumber: "+573001234567", DisplayName: "Juan"}}}
	svc.convRepo = &mockConvRepo{byContact: map[int32]*domain.Conversation{11: {ID: 3, ContactID: 11}}}
}

func TestNotify_RealLinkerAppendsTrackedLink(t *testing.T) {
	repo := newMockInvoiceRepo()
	provider := &mockProvider{
		customer: &invdomain.CustomerRef{ExternalID: "cust-1"},
		created: &invdomain.Invoice{
			OrganizationID: 7, DealID: 99, ExternalID: "inv-1",
			Status: invdomain.InvoiceStatusPending,
			Amount: float64Ptr(250000),
		},
	}
	svc, out := newTestService(repo, provider)
	setupNotifyEnv(svc)

	linker := &fakeLinker{link: "https://checkout.mercadopago.com/x"}
	svc.paymentLinker = linker

	inv, err := svc.CreateForDeal(context.Background(), 7, 99)
	if err != nil {
		t.Fatalf("CreateForDeal failed: %v", err)
	}

	if linker.calls != 1 {
		t.Fatalf("expected 1 linker call, got %d", linker.calls)
	}
	if linker.orgID != 7 || linker.dealID != 99 || linker.amountCOP != 250000 {
		t.Fatalf("linker called with wrong args: org=%d deal=%d amount=%d", linker.orgID, linker.dealID, linker.amountCOP)
	}
	if len(out.sent) != 1 || !strings.Contains(out.sent[0], "https://checkout.mercadopago.com/x") {
		t.Fatalf("notification must contain the payment link, got %v", out.sent)
	}
	_ = inv
}

func TestNotify_LinkerFailureIsNonFatal(t *testing.T) {
	repo := newMockInvoiceRepo()
	provider := &mockProvider{
		customer: &invdomain.CustomerRef{ExternalID: "cust-1"},
		created: &invdomain.Invoice{
			OrganizationID: 7, DealID: 99, ExternalID: "inv-1",
			Status: invdomain.InvoiceStatusPending,
			Amount: float64Ptr(100000),
		},
	}
	svc, out := newTestService(repo, provider)
	setupNotifyEnv(svc)

	svc.paymentLinker = &fakeLinker{err: errors.New("mp down")}

	if _, err := svc.CreateForDeal(context.Background(), 7, 99); err != nil {
		t.Fatalf("linker failure must not fail invoicing: %v", err)
	}
	if len(out.sent) != 1 || strings.Contains(out.sent[0], "https://") {
		t.Fatalf("notification must still be sent without the link, got %v", out.sent)
	}
}

func TestNotify_NilLinkerSendsWithoutLink(t *testing.T) {
	repo := newMockInvoiceRepo()
	provider := &mockProvider{
		customer: &invdomain.CustomerRef{ExternalID: "cust-1"},
		created: &invdomain.Invoice{
			OrganizationID: 7, DealID: 99, ExternalID: "inv-1",
			Status: invdomain.InvoiceStatusPending,
			Amount: float64Ptr(100000),
		},
	}
	svc, out := newTestService(repo, provider)
	setupNotifyEnv(svc)

	if _, err := svc.CreateForDeal(context.Background(), 7, 99); err != nil {
		t.Fatalf("CreateForDeal failed: %v", err)
	}
	if len(out.sent) != 1 || strings.Contains(out.sent[0], "Pagar") {
		t.Fatalf("no linker → no payment section, got %v", out.sent)
	}
}

func float64Ptr(v float64) *float64 { return &v }
