package routing

import (
	"context"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
)

type BillingProviderResolver interface {
	GetBillingProvider(ctx context.Context, organizationID int32) (string, error)
}

type ProviderRouter struct {
	polarAdapter domain.BillingProvider
	mpAdapter    domain.BillingProvider
	resolver     BillingProviderResolver
	orgAdapter   domain.OrganizationAdapter
}

func NewProviderRouter(
	polarAdapter domain.BillingProvider,
	mpAdapter domain.BillingProvider,
	resolver BillingProviderResolver,
	orgAdapter domain.OrganizationAdapter,
) domain.BillingProvider {
	return &ProviderRouter{
		polarAdapter: polarAdapter,
		mpAdapter:    mpAdapter,
		resolver:     resolver,
		orgAdapter:   orgAdapter,
	}
}

func (r *ProviderRouter) resolveProvider(ctx context.Context, orgID int32) (domain.BillingProvider, error) {
	provider, err := r.resolver.GetBillingProvider(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve billing provider: %w", err)
	}

	switch provider {
	case "", "polar":
		return r.polarAdapter, nil
	case "mercadopago":
		return r.mpAdapter, nil
	default:
		return nil, fmt.Errorf("unsupported billing provider: %s", provider)
	}
}

func (r *ProviderRouter) GetSubscription(ctx context.Context, externalCustomerID string) (*domain.Subscription, error) {
	orgID, err := r.orgAdapter.GetOrganizationIDByStytchOrgID(ctx, externalCustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization from external customer ID: %w", err)
	}
	adapter, err := r.resolveProvider(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return adapter.GetSubscription(ctx, externalCustomerID)
}

func (r *ProviderRouter) GetCheckoutSession(ctx context.Context, sessionID string) (*domain.CheckoutSessionResponse, error) {
	return r.polarAdapter.GetCheckoutSession(ctx, sessionID)
}

func (r *ProviderRouter) GetCheckoutSessionWithPolling(ctx context.Context, sessionID string) (*domain.CheckoutSessionResponse, error) {
	return r.polarAdapter.GetCheckoutSessionWithPolling(ctx, sessionID)
}

func (r *ProviderRouter) IngestMeterEvent(ctx context.Context, externalCustomerID string, meterSlug string, amount int32) error {
	orgID, err := r.orgAdapter.GetOrganizationIDByStytchOrgID(ctx, externalCustomerID)
	if err != nil {
		return fmt.Errorf("failed to get organization from external customer ID: %w", err)
	}
	adapter, err := r.resolveProvider(ctx, orgID)
	if err != nil {
		return err
	}
	return adapter.IngestMeterEvent(ctx, externalCustomerID, meterSlug, amount)
}
