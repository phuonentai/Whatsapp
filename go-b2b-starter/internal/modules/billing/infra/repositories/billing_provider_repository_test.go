package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
)

// fakeProviderStore embeds the full sqlc.Store interface so it satisfies the
// interface regardless of added queries; only the methods under test are
// overridden. Embedding with a nil embedded field panics only if an
// unimplemented method is invoked, which the tested code path never does.
type fakeProviderStore struct {
	sqlc.Store
	providers map[int32]string
	err       error
}

func (f *fakeProviderStore) GetOrganizationBillingProvider(ctx context.Context, id int32) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.providers[id], nil
}

func (f *fakeProviderStore) SetOrganizationBillingProvider(ctx context.Context, arg sqlc.SetOrganizationBillingProviderParams) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.providers[arg.ID] = arg.BillingProvider.String
	return arg.BillingProvider.String, nil
}

func TestBillingProviderResolver_DefaultsToPolar(t *testing.T) {
	resolver := NewBillingProviderResolver(&fakeProviderStore{providers: map[int32]string{}})

	provider, err := resolver.GetBillingProvider(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "polar", provider)
}

func TestBillingProviderResolver_ReturnsStoredProvider(t *testing.T) {
	resolver := NewBillingProviderResolver(&fakeProviderStore{providers: map[int32]string{1: "mercadopago"}})

	provider, err := resolver.GetBillingProvider(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "mercadopago", provider)
}

func TestBillingProviderResolver_SetsProvider(t *testing.T) {
	store := &fakeProviderStore{providers: map[int32]string{}}
	resolver := NewBillingProviderResolver(store)

	err := resolver.SetBillingProvider(context.Background(), 1, "mercadopago")
	require.NoError(t, err)
	assert.Equal(t, "mercadopago", store.providers[1])

	provider, err := resolver.GetBillingProvider(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "mercadopago", provider)
}

func TestBillingProviderResolver_PropagatesStoreError(t *testing.T) {
	resolver := NewBillingProviderResolver(&fakeProviderStore{err: errors.New("db down")})

	_, err := resolver.GetBillingProvider(context.Background(), 1)
	require.Error(t, err)

	err = resolver.SetBillingProvider(context.Background(), 1, "mercadopago")
	require.Error(t, err)
}
