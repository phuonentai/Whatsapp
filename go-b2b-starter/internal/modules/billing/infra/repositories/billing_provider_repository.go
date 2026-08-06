package repositories

import (
	"context"

	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/billing/infra/routing"
)

type billingProviderResolver struct {
	store sqlc.Store
}

func NewBillingProviderResolver(store sqlc.Store) routing.BillingProviderResolver {
	return &billingProviderResolver{store: store}
}

func (r *billingProviderResolver) GetBillingProvider(_ context.Context, _ int32) (string, error) {
	return "polar", nil
}
