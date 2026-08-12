package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
)

// fakeMPProvider is a minimal MercadoPago provider double for the MP-specific
// service methods. Embedding the interface keeps unrelated methods satisfied.
type fakeMPProvider struct {
	domain.BillingProvider
	cancelErr   error
	sub         *domain.Subscription
	checkout    *domain.CheckoutSessionResponse
	checkoutErr error
}

func (f *fakeMPProvider) CancelSubscription(ctx context.Context, subscriptionID string) error {
	return f.cancelErr
}

func (f *fakeMPProvider) GetSubscription(ctx context.Context, externalCustomerID string) (*domain.Subscription, error) {
	if f.sub == nil {
		return nil, domain.ErrSubscriptionNotFound
	}
	return f.sub, nil
}

func (f *fakeMPProvider) CreateCheckoutSession(ctx context.Context, planID, externalReference string) (*domain.CheckoutSessionResponse, error) {
	if f.checkoutErr != nil {
		return nil, f.checkoutErr
	}
	if f.checkout != nil {
		return f.checkout, nil
	}
	return &domain.CheckoutSessionResponse{ID: "pre-1", Status: "authorized", InitPoint: "https://checkout.example/x"}, nil
}

func (f *fakeMPProvider) GetCheckoutSessionWithPolling(ctx context.Context, sessionID string) (*domain.CheckoutSessionResponse, error) {
	if f.checkout != nil {
		return f.checkout, nil
	}
	return &domain.CheckoutSessionResponse{ID: sessionID, Status: "succeeded", CustomerID: "org_stytch_1"}, nil
}

// fakeMPResolver records SetBillingProvider calls.
type fakeMPResolver struct {
	providers map[int32]string
}

func (f *fakeMPResolver) GetBillingProvider(ctx context.Context, organizationID int32) (string, error) {
	return f.providers[organizationID], nil
}

func (f *fakeMPResolver) SetBillingProvider(ctx context.Context, organizationID int32, provider string) error {
	if f.providers == nil {
		f.providers = make(map[int32]string)
	}
	f.providers[organizationID] = provider
	return nil
}

func newMPCheckoutService(repo *fakeUpsertRepo, mpProvider domain.BillingProvider) *billingService {
	return &billingService{
		repo:       repo,
		orgAdapter: &fakeGrantOrgAdapter{orgByStytch: map[string]int32{"org_stytch_1": 42}},
		mpProvider: mpProvider,
		resolver:   &fakeMPResolver{},
		logger:     fakeGrantLogger{},
	}
}

func TestCreateMPCheckout_UnconfiguredProviderReturnsClearError(t *testing.T) {
	svc := newMPCheckoutService(&fakeUpsertRepo{}, nil)

	_, err := svc.CreateMPCheckout(context.Background(), "org_stytch_1", "plan-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mercadopago not configured")
}

func TestVerifyMPPayment_UnconfiguredProviderReturnsClearError(t *testing.T) {
	svc := newMPCheckoutService(&fakeUpsertRepo{}, nil)

	_, err := svc.VerifyMPPayment(context.Background(), "pay-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mercadopago not configured")
}

func TestCancelMPSubscription_UnconfiguredProviderReturnsClearError(t *testing.T) {
	svc := newMPCheckoutService(&fakeUpsertRepo{}, nil)

	_, err := svc.CancelMPSubscription(context.Background(), "org_stytch_1", "pre-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mercadopago not configured")
}

func TestCancelMPSubscription_MissingOrgContextErrors(t *testing.T) {
	svc := newMPCheckoutService(&fakeUpsertRepo{}, &fakeMPProvider{})

	_, err := svc.CancelMPSubscription(context.Background(), "", "pre-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "organization context required")
}

func TestCreateMPCheckout_ResolvesOrgFromExplicitParam(t *testing.T) {
	repo := &fakeUpsertRepo{}
	svc := newMPCheckoutService(repo, &fakeMPProvider{})

	status, err := svc.CreateMPCheckout(context.Background(), "org_stytch_1", "plan-1")
	require.NoError(t, err)
	assert.Equal(t, int32(42), status.OrganizationID)
	assert.Equal(t, "org_stytch_1", status.ExternalID)
	assert.Equal(t, "https://checkout.example/x", status.CheckoutURL)
}

func TestCancelMPSubscription_PersistsCanceledRowWithExistingPeriodBounds(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := &fakeUpsertRepo{stored: &domain.Subscription{
		OrganizationID:     42,
		ExternalCustomerID: "org_stytch_1",
		SubscriptionID:     "pre-1",
		SubscriptionStatus: "active",
		CurrentPeriodStart: start,
		CurrentPeriodEnd:   end,
	}}
	svc := newMPCheckoutService(repo, &fakeMPProvider{})

	status, err := svc.CancelMPSubscription(context.Background(), "org_stytch_1", "pre-1")
	require.NoError(t, err)
	assert.False(t, status.HasActiveSubscription)

	require.NotNil(t, repo.stored)
	assert.Equal(t, "canceled", repo.stored.SubscriptionStatus)
	assert.Equal(t, start, repo.stored.CurrentPeriodStart)
	assert.Equal(t, end, repo.stored.CurrentPeriodEnd)
}

func TestCancelMPSubscription_FallsBackToProviderPeriodBounds(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := &fakeUpsertRepo{} // no existing row
	provider := &fakeMPProvider{sub: &domain.Subscription{
		SubscriptionID:     "pre-1",
		CurrentPeriodStart: start,
		CurrentPeriodEnd:   end,
	}}
	svc := newMPCheckoutService(repo, provider)

	status, err := svc.CancelMPSubscription(context.Background(), "org_stytch_1", "pre-1")
	require.NoError(t, err)
	assert.False(t, status.HasActiveSubscription)

	require.NotNil(t, repo.stored)
	assert.Equal(t, "canceled", repo.stored.SubscriptionStatus)
	assert.Equal(t, start, repo.stored.CurrentPeriodStart)
	assert.Equal(t, end, repo.stored.CurrentPeriodEnd)
}

func TestCancelMPSubscription_NeverZeroPeriodBounds(t *testing.T) {
	repo := &fakeUpsertRepo{} // no existing row, provider has none either
	svc := newMPCheckoutService(repo, &fakeMPProvider{})

	_, err := svc.CancelMPSubscription(context.Background(), "org_stytch_1", "pre-1")
	require.NoError(t, err)

	require.NotNil(t, repo.stored)
	assert.False(t, repo.stored.CurrentPeriodStart.IsZero(), "NOT NULL current_period_start must never be zero")
	assert.False(t, repo.stored.CurrentPeriodEnd.IsZero(), "NOT NULL current_period_end must never be zero")
}

func TestVerifyMPPayment_FetchesFromMPProviderAndSetsProviderLast(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := &fakeUpsertRepo{}
	provider := &fakeMPProvider{checkout: &domain.CheckoutSessionResponse{
		ID:         "pay-1",
		Status:     "succeeded",
		CustomerID: "org_stytch_1",
	}}
	provider.sub = &domain.Subscription{
		ExternalCustomerID: "org_stytch_1",
		SubscriptionID:     "pre-1",
		SubscriptionStatus: "active",
		CurrentPeriodStart: start,
		CurrentPeriodEnd:   end,
		Metadata:           map[string]any{"invoice_count_max": int32(25)},
	}
	svc := newMPCheckoutService(repo, provider)

	status, err := svc.VerifyMPPayment(context.Background(), "pay-1")
	require.NoError(t, err)
	assert.True(t, status.HasActiveSubscription)
	assert.True(t, status.CanProcessInvoices)
	assert.Equal(t, int32(25), status.InvoiceCount)
	require.NotNil(t, repo.stored)
	assert.Equal(t, "active", repo.stored.SubscriptionStatus)
	require.NotNil(t, repo.quota)
	assert.Equal(t, int32(25), repo.quota.InvoiceCount)
	// SetBillingProvider must run last: the org is recorded as mercadopago
	// only after the subscription/quota upserts succeed.
	resolver := svc.resolver.(*fakeMPResolver)
	assert.Equal(t, "mercadopago", resolver.providers[42])
}
