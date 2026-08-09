package routing

import (
	"context"
	"errors"
	"testing"

	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
)

type fakeAdapter struct {
	calls int
}

func (f *fakeAdapter) CreateInvoice(ctx context.Context, orgID int32, req *domain.InvoiceRequest) (*domain.Invoice, error) {
	f.calls++
	return &domain.Invoice{OrganizationID: orgID, DealID: req.DealID, Status: domain.InvoiceStatusPending}, nil
}

func (f *fakeAdapter) GetInvoiceStatus(ctx context.Context, orgID int32, externalID string) (*domain.Invoice, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeAdapter) UpsertCustomer(ctx context.Context, orgID int32, customer domain.CustomerInfo) (*domain.CustomerRef, error) {
	return nil, errors.New("not implemented")
}

func TestInvoiceRouter_DelegatesToSiigo(t *testing.T) {
	adapter := &fakeAdapter{}
	router := NewInvoiceRouter(adapter, NewStaticResolver("siigo"))

	inv, err := router.CreateInvoice(context.Background(), 5, &domain.InvoiceRequest{DealID: 1})
	if err != nil {
		t.Fatalf("CreateInvoice failed: %v", err)
	}
	if adapter.calls != 1 {
		t.Fatalf("expected 1 adapter call, got %d", adapter.calls)
	}
	if inv.DealID != 1 {
		t.Fatalf("unexpected deal id %d", inv.DealID)
	}
}

func TestInvoiceRouter_FailsClosedOnUnknownProvider(t *testing.T) {
	adapter := &fakeAdapter{}
	router := NewInvoiceRouter(adapter, NewStaticResolver("alegra"))

	if _, err := router.CreateInvoice(context.Background(), 5, &domain.InvoiceRequest{DealID: 1}); err == nil {
		t.Fatal("expected error for unknown provider")
	}
}
