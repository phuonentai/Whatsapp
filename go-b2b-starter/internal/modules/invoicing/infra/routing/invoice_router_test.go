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

func TestInvoiceRouter_NoneProviderIsNoop(t *testing.T) {
	adapter := &fakeAdapter{}
	router := NewInvoiceRouter(adapter, NewStaticResolver("none"))

	inv, err := router.CreateInvoice(context.Background(), 5, &domain.InvoiceRequest{DealID: 1})
	if err != nil {
		t.Fatalf("none provider must not error: %v", err)
	}
	if inv != nil {
		t.Fatalf("expected nil invoice from noop provider, got %+v", inv)
	}
	if adapter.calls != 0 {
		t.Fatalf("expected 0 adapter calls for none provider, got %d", adapter.calls)
	}

	cust, err := router.UpsertCustomer(context.Background(), 5, domain.CustomerInfo{Name: "X"})
	if err != nil || cust != nil {
		t.Fatalf("upsert on none provider must be a noop, got %+v, %v", cust, err)
	}

	status, err := router.GetInvoiceStatus(context.Background(), 5, "ext-1")
	if err != nil || status != nil {
		t.Fatalf("status on none provider must be a noop, got %+v, %v", status, err)
	}
}
