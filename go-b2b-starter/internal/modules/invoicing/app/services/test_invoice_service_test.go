package services

import (
	"context"
	"errors"
	"testing"

	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
)

type fakeTestInvoiceProvider struct {
	created  *domain.Invoice
	status   *domain.Invoice
	statusErr error
}

func (f *fakeTestInvoiceProvider) CreateInvoice(ctx context.Context, orgID int32, req *domain.InvoiceRequest) (*domain.Invoice, error) {
	return f.created, nil
}

func (f *fakeTestInvoiceProvider) GetInvoiceStatus(ctx context.Context, orgID int32, externalID string) (*domain.Invoice, error) {
	return f.status, f.statusErr
}

func (f *fakeTestInvoiceProvider) UpsertCustomer(ctx context.Context, orgID int32, customer domain.CustomerInfo) (*domain.CustomerRef, error) {
	return nil, nil
}

type fakeTestConnSvc struct {
	sandboxOK int
	err       error
}

func (f *fakeTestConnSvc) Status(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	return nil, nil
}
func (f *fakeTestConnSvc) Connect(ctx context.Context, orgID int32, req ConnectRequest) (*domain.OrgConnection, error) {
	return nil, nil
}
func (f *fakeTestConnSvc) RequestAssisted(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	return nil, nil
}
func (f *fakeTestConnSvc) Provision(ctx context.Context, orgID int32, req ConnectRequest) (*domain.OrgConnection, error) {
	return nil, nil
}
func (f *fakeTestConnSvc) Pause(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	return nil, nil
}
func (f *fakeTestConnSvc) Resume(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	return nil, nil
}
func (f *fakeTestConnSvc) Activate(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	return nil, nil
}
func (f *fakeTestConnSvc) Disable(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	return nil, nil
}
func (f *fakeTestConnSvc) ConfirmNumeration(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	return nil, nil
}
func (f *fakeTestConnSvc) ConfirmSandboxOK(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	f.sandboxOK++
	return nil, f.err
}
func (f *fakeTestConnSvc) IsLive(ctx context.Context, orgID int32) (bool, error) {
	return false, nil
}
func (f *fakeTestConnSvc) StatusAll(ctx context.Context) ([]*domain.OrgConnection, error) {
	return nil, nil
}

func newTestInvoiceServiceForTest(t *testing.T, provider domain.InvoicingProvider, connSvc ConnectionService) TestInvoiceService {
	t.Helper()
	repo := newMockInvoiceRepo()
	return NewTestInvoiceService(provider, repo, connSvc, nopLogger{})
}

func TestTestInvoice_PendingStatusDoesNotAdvance(t *testing.T) {
	created := &domain.Invoice{OrganizationID: 7, ExternalID: "inv-test-1", Status: domain.InvoiceStatusPending}
	provider := &fakeTestInvoiceProvider{
		created: created,
		status:  &domain.Invoice{ExternalID: "inv-test-1", Status: domain.InvoiceStatusPending},
	}
	connSvc := &fakeTestConnSvc{}
	svc := newTestInvoiceServiceForTest(t, provider, connSvc)

	inv, err := svc.CreateTestInvoice(context.Background(), 7)
	if err != nil {
		t.Fatalf("test invoice failed: %v", err)
	}
	if inv.Status != domain.InvoiceStatusPending {
		t.Fatalf("expected pending status, got %s", inv.Status)
	}
	if connSvc.sandboxOK != 0 {
		t.Fatalf("pending status must not advance to sandbox_ok, got %d calls", connSvc.sandboxOK)
	}
}

func TestTestInvoice_ValidStatusAdvancesSandboxOK(t *testing.T) {
	created := &domain.Invoice{OrganizationID: 7, ExternalID: "inv-test-2", Status: domain.InvoiceStatusPending}
	provider := &fakeTestInvoiceProvider{
		created: created,
		status:  &domain.Invoice{ExternalID: "inv-test-2", Status: domain.InvoiceStatusValid, Cufe: "CUFE1", PdfURL: "https://pdf"},
	}
	connSvc := &fakeTestConnSvc{}
	svc := newTestInvoiceServiceForTest(t, provider, connSvc)

	inv, err := svc.CreateTestInvoice(context.Background(), 7)
	if err != nil {
		t.Fatalf("test invoice failed: %v", err)
	}
	if inv.Status != domain.InvoiceStatusValid {
		t.Fatalf("expected valid status, got %s", inv.Status)
	}
	if connSvc.sandboxOK != 1 {
		t.Fatalf("expected 1 sandbox_ok advance, got %d", connSvc.sandboxOK)
	}
}

func TestTestInvoice_StatusCheckFailureIsNonFatal(t *testing.T) {
	created := &domain.Invoice{OrganizationID: 7, ExternalID: "inv-test-3", Status: domain.InvoiceStatusPending}
	provider := &fakeTestInvoiceProvider{
		created:   created,
		statusErr: errors.New("provider down"),
	}
	connSvc := &fakeTestConnSvc{}
	svc := newTestInvoiceServiceForTest(t, provider, connSvc)

	inv, err := svc.CreateTestInvoice(context.Background(), 7)
	if err != nil {
		t.Fatalf("status-check failure must not fail the test invoice: %v", err)
	}
	if inv.Status != domain.InvoiceStatusPending {
		t.Fatalf("expected stored pending status, got %s", inv.Status)
	}
	if connSvc.sandboxOK != 0 {
		t.Fatalf("no advance allowed on failed status check, got %d", connSvc.sandboxOK)
	}
}

func TestTestInvoice_NoopProviderNilResultIsErrorNotPanic(t *testing.T) {
	provider := &fakeTestInvoiceProvider{created: nil} // provider returns nil, nil
	connSvc := &fakeTestConnSvc{}
	svc := newTestInvoiceServiceForTest(t, provider, connSvc)

	_, err := svc.CreateTestInvoice(context.Background(), 7)
	if err == nil {
		t.Fatal("expected error when provider returns nil invoice")
	}
	if connSvc.sandboxOK != 0 {
		t.Fatalf("no advance allowed on nil invoice, got %d", connSvc.sandboxOK)
	}
}

func TestTestInvoice_AlreadyValidOnCreateStillAdvances(t *testing.T) {
	// Provider resolves valid on POST (mock behavior): the stored invoice is
	// already valid, but sandbox_ok MUST still advance — a same-status early
	// return would dead-end the onboarding flow.
	created := &domain.Invoice{OrganizationID: 7, ExternalID: "inv-test-4", Status: domain.InvoiceStatusValid, Cufe: "C", PdfURL: "p"}
	provider := &fakeTestInvoiceProvider{
		created: created,
		status:  &domain.Invoice{ExternalID: "inv-test-4", Status: domain.InvoiceStatusValid, Cufe: "C", PdfURL: "p"},
	}
	connSvc := &fakeTestConnSvc{}
	svc := newTestInvoiceServiceForTest(t, provider, connSvc)

	inv, err := svc.CreateTestInvoice(context.Background(), 7)
	if err != nil {
		t.Fatalf("test invoice failed: %v", err)
	}
	if inv.Status != domain.InvoiceStatusValid {
		t.Fatalf("expected valid, got %s", inv.Status)
	}
	if connSvc.sandboxOK != 1 {
		t.Fatalf("valid-on-create must still advance to sandbox_ok, got %d calls", connSvc.sandboxOK)
	}
}
