package routing

import (
	"context"

	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
)

// NoopProvider is the explicit "no invoicing provider" state. Organizations
// without a provider route here: no provider call is made and no error is
// returned — deal-stage gating handles the user-visible behaviour. This is a
// deliberate state, not a fail-closed error (unknown providers still fail
// closed).
type NoopProvider struct{}

func NewNoopProvider() domain.InvoicingProvider { return &NoopProvider{} }

func (p *NoopProvider) CreateInvoice(ctx context.Context, orgID int32, req *domain.InvoiceRequest) (*domain.Invoice, error) {
	return nil, nil
}

func (p *NoopProvider) GetInvoiceStatus(ctx context.Context, orgID int32, externalID string) (*domain.Invoice, error) {
	return nil, nil
}

func (p *NoopProvider) UpsertCustomer(ctx context.Context, orgID int32, customer domain.CustomerInfo) (*domain.CustomerRef, error) {
	return nil, nil
}
