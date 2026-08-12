package routing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
)

type fakeAdapter struct {
	name       string
	subscription *domain.Subscription
}

func (f *fakeAdapter) GetSubscription(ctx context.Context, externalCustomerID string) (*domain.Subscription, error) {
	if f.subscription == nil {
		return nil, domain.ErrSubscriptionNotFound
	}
	return f.subscription, nil
}

func (f *fakeAdapter) GetCheckoutSession(ctx context.Context, sessionID string) (*domain.CheckoutSessionResponse, error) {
	return &domain.CheckoutSessionResponse{ID: sessionID, Status: "succeeded"}, nil
}

func (f *fakeAdapter) GetCheckoutSessionWithPolling(ctx context.Context, sessionID string) (*domain.CheckoutSessionResponse, error) {
	return &domain.CheckoutSessionResponse{ID: sessionID, Status: "succeeded"}, nil
}

func (f *fakeAdapter) IngestMeterEvent(ctx context.Context, externalCustomerID string, meterSlug string, amount int32) error {
	return nil
}

type fakeResolver struct {
	providers map[int32]string
}

func (f *fakeResolver) GetBillingProvider(ctx context.Context, organizationID int32) (string, error) {
	if p, ok := f.providers[organizationID]; ok {
		return p, nil
	}
	return "polar", nil
}

func (f *fakeResolver) SetBillingProvider(ctx context.Context, organizationID int32, provider string) error {
	f.providers[organizationID] = provider
	return nil
}

type fakeOrgAdapter struct{}

func (f *fakeOrgAdapter) GetStytchOrgID(ctx context.Context, organizationID int32) (string, error) {
	return "org-stytch", nil
}

func (f *fakeOrgAdapter) GetOrganizationIDByStytchOrgID(ctx context.Context, stytchOrgID string) (int32, error) {
	return 1, nil
}

func newTestRouter() *ProviderRouter {
	router := NewProviderRouter(
		&fakeAdapter{name: "polar", subscription: &domain.Subscription{SubscriptionID: "pol_1"}},
		&fakeAdapter{name: "mercadopago", subscription: &domain.Subscription{SubscriptionID: "mp_1"}},
		&fakeResolver{providers: map[int32]string{1: "polar", 2: "mercadopago", 3: "bogus"}},
		&fakeOrgAdapter{},
	)
	return router.(*ProviderRouter)
}

func TestProviderRouter_RoutesToPolar(t *testing.T) {
	router := newTestRouter()

	sub, err := router.GetSubscription(context.Background(), "org-stytch")
	require.NoError(t, err)
	assert.Equal(t, "pol_1", sub.SubscriptionID)
}

func TestProviderRouter_RoutesToMercadoPago(t *testing.T) {
	router := newTestRouter()
	orgAdapter := &fakeOrgAdapter{}
	_ = orgAdapter

	// override org mapping to resolve org 2 (mercadopago)
	router.orgAdapter = &fixedOrgAdapter{orgID: 2}

	sub, err := router.GetSubscription(context.Background(), "org-stytch")
	require.NoError(t, err)
	assert.Equal(t, "mp_1", sub.SubscriptionID)
}

type fixedOrgAdapter struct{ orgID int32 }

func (f *fixedOrgAdapter) GetStytchOrgID(ctx context.Context, organizationID int32) (string, error) {
	return "org-stytch", nil
}

func (f *fixedOrgAdapter) GetOrganizationIDByStytchOrgID(ctx context.Context, stytchOrgID string) (int32, error) {
	return f.orgID, nil
}

func TestProviderRouter_UnconfiguredMPDelegatesAllOrgsToPolar(t *testing.T) {
	// MercadoPago is optional in DI: when unconfigured the router receives a
	// nil MP adapter and MUST degrade to Polar-only routing for every org,
	// including ones whose provider is recorded as "mercadopago".
	router := NewProviderRouter(
		&fakeAdapter{name: "polar", subscription: &domain.Subscription{SubscriptionID: "pol_1"}},
		nil,
		&fakeResolver{providers: map[int32]string{1: "polar", 2: "mercadopago", 3: ""}},
		&fakeOrgAdapter{},
	).(*ProviderRouter)

	for _, orgID := range []int32{1, 2, 3} {
		router.orgAdapter = &fixedOrgAdapter{orgID: orgID}
		sub, err := router.GetSubscription(context.Background(), "org-stytch")
		require.NoError(t, err)
		assert.Equal(t, "pol_1", sub.SubscriptionID, "org %d must delegate to Polar when MP is unconfigured", orgID)
	}
}

func TestProviderRouter_UnknownProviderErrors(t *testing.T) {
	router := newTestRouter()
	router.orgAdapter = &fixedOrgAdapter{orgID: 3}

	_, err := router.GetSubscription(context.Background(), "org-stytch")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported billing provider")
}

func TestProviderRouter_CheckoutSessionDelegatesToPolar(t *testing.T) {
	router := newTestRouter()

	// Checkout-session methods are Polar-checkout specific; MP payment
	// verification uses the MP adapter directly in the service layer.
	session, err := router.GetCheckoutSessionWithPolling(context.Background(), "sess_1")
	require.NoError(t, err)
	assert.Equal(t, "succeeded", session.Status)
}

func TestProviderRouter_PropagatesOrgLookupError(t *testing.T) {
	router := newTestRouter()
	router.orgAdapter = &errOrgAdapter{err: errors.New("org lookup failed")}

	_, err := router.GetSubscription(context.Background(), "org-stytch")
	require.Error(t, err)
}

type errOrgAdapter struct{ err error }

func (f *errOrgAdapter) GetStytchOrgID(ctx context.Context, organizationID int32) (string, error) {
	return "", f.err
}

func (f *errOrgAdapter) GetOrganizationIDByStytchOrgID(ctx context.Context, stytchOrgID string) (int32, error) {
	return 0, f.err
}
