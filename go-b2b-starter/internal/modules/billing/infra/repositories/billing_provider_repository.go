package repositories

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/billing/infra/routing"
)

// providerStore narrows the sqlc.Store surface to the queries the resolver
// needs, keeping it easy to fake in tests while remaining satisfied by
// *sqlc.SQLStore.
type providerStore interface {
	GetOrganizationBillingProvider(ctx context.Context, id int32) (string, error)
	SetOrganizationBillingProvider(ctx context.Context, arg sqlc.SetOrganizationBillingProviderParams) (string, error)
}

type billingProviderResolver struct {
	store providerStore
}

func NewBillingProviderResolver(store sqlc.Store) routing.BillingProviderResolver {
	return &billingProviderResolver{store: store}
}

func (r *billingProviderResolver) GetBillingProvider(ctx context.Context, organizationID int32) (string, error) {
	provider, err := r.store.GetOrganizationBillingProvider(ctx, organizationID)
	if err != nil {
		return "", fmt.Errorf("failed to resolve billing provider for organization %d: %w", organizationID, err)
	}
	if provider == "" {
		return "polar", nil
	}
	return provider, nil
}

func (r *billingProviderResolver) SetBillingProvider(ctx context.Context, organizationID int32, provider string) error {
	_, err := r.store.SetOrganizationBillingProvider(ctx, sqlc.SetOrganizationBillingProviderParams{
		ID:              organizationID,
		BillingProvider: pgtype.Text{String: provider, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to set billing provider for organization %d: %w", organizationID, err)
	}
	return nil
}
