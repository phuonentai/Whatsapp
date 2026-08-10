// Package routing resolves the per-organization invoicing provider, mirroring
// the billing ProviderRouter pattern (billing/infra/routing/provider_router.go).
package routing

import (
	"context"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
)

// ProviderResolver resolves the invoicing provider for an organization.
// Returns "siigo" by default; a future DB-backed implementation can route
// per-org (e.g. to Alegra) without touching the router.
type ProviderResolver interface {
	GetInvoicingProvider(ctx context.Context, organizationID int32) (string, error)
}

// StaticResolver returns a constant provider per org (default "siigo").
type StaticResolver struct{ provider string }

func NewStaticResolver(provider string) ProviderResolver {
	return &StaticResolver{provider: provider}
}

func (r *StaticResolver) GetInvoicingProvider(ctx context.Context, organizationID int32) (string, error) {
	return r.provider, nil
}

// InvoiceRouter implements domain.InvoicingProvider and delegates to the
// adapter resolved for the organization. Siigo is the only adapter today; the
// resolver + named-adapter seam allows a second provider (Alegra) to be added
// without changing the invoicing service.
type InvoiceRouter struct {
	siigoAdapter domain.InvoicingProvider
	noopProvider domain.InvoicingProvider
	resolver     ProviderResolver
}

func NewInvoiceRouter(siigoAdapter domain.InvoicingProvider, resolver ProviderResolver) domain.InvoicingProvider {
	return &InvoiceRouter{siigoAdapter: siigoAdapter, noopProvider: NewNoopProvider(), resolver: resolver}
}

func (r *InvoiceRouter) resolveProvider(ctx context.Context, orgID int32) (domain.InvoicingProvider, error) {
	provider, err := r.resolver.GetInvoicingProvider(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve invoicing provider: %w", err)
	}
	switch provider {
	case "", "siigo":
		return r.siigoAdapter, nil
	case "none":
		// Explicit no-provider state: no-op, not an error.
		return r.noopProvider, nil
	default:
		return nil, fmt.Errorf("unsupported invoicing provider: %s", provider)
	}
}

func (r *InvoiceRouter) CreateInvoice(ctx context.Context, orgID int32, req *domain.InvoiceRequest) (*domain.Invoice, error) {
	adapter, err := r.resolveProvider(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return adapter.CreateInvoice(ctx, orgID, req)
}

func (r *InvoiceRouter) GetInvoiceStatus(ctx context.Context, orgID int32, externalID string) (*domain.Invoice, error) {
	adapter, err := r.resolveProvider(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return adapter.GetInvoiceStatus(ctx, orgID, externalID)
}

func (r *InvoiceRouter) UpsertCustomer(ctx context.Context, orgID int32, customer domain.CustomerInfo) (*domain.CustomerRef, error) {
	adapter, err := r.resolveProvider(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return adapter.UpsertCustomer(ctx, orgID, customer)
}
