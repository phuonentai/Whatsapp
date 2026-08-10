package routing

import (
	"context"
	"errors"

	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
)

// ConnectionProviderResolver derives the per-organization invoicing provider
// from the org connection state: only live connections route to their
// provider ("siigo"); any other state routes to the explicit "none" no-op
// provider. Unknown/unrecognized states never resolve to a concrete adapter.
type ConnectionProviderResolver struct {
	repo domain.ConnectionRepository
}

func NewConnectionProviderResolver(repo domain.ConnectionRepository) ProviderResolver {
	return &ConnectionProviderResolver{repo: repo}
}

func (r *ConnectionProviderResolver) GetInvoicingProvider(ctx context.Context, organizationID int32) (string, error) {
	conn, err := r.repo.Get(ctx, organizationID)
	if err != nil {
		if errors.Is(err, domain.ErrConnectionNotFound) {
			return "none", nil
		}
		return "", err
	}
	if !conn.Status.IsLive() {
		return "none", nil
	}
	return conn.Provider, nil
}
