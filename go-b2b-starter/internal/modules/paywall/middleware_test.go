package paywall

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
)

// fakeProvider implements SubscriptionStatusProvider for middleware tests.
type fakeProvider struct {
	getStatus     *SubscriptionStatus
	getErr        error
	refreshStatus *SubscriptionStatus
	refreshErr    error
	refreshCalls  int
}

func (f *fakeProvider) GetSubscriptionStatus(ctx context.Context, organizationID int32) (*SubscriptionStatus, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.getStatus, nil
}

func (f *fakeProvider) RefreshSubscriptionStatus(ctx context.Context, organizationID int32) (*SubscriptionStatus, error) {
	f.refreshCalls++
	if f.refreshErr != nil {
		return nil, f.refreshErr
	}
	return f.refreshStatus, nil
}

// withOrgContext simulates auth.RequireOrganization having run.
func withOrgContext(orgID int32) gin.HandlerFunc {
	return func(c *gin.Context) {
		authcontext.SetRequestContext(c, &authcontext.RequestContext{OrganizationID: orgID})
		c.Next()
	}
}

func doRequest(provider SubscriptionStatusProvider, handler gin.HandlerFunc) (*httptest.ResponseRecorder, *gin.Context) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var captured *gin.Context
	r.GET("/test", withOrgContext(42), handler, func(c *gin.Context) {
		captured = c
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))
	return w, captured
}

func TestUnknownStatusTriggersRefreshBeforeDenying(t *testing.T) {
	provider := &fakeProvider{
		getStatus:     &SubscriptionStatus{OrganizationID: 42, IsActive: false, Status: StatusUnknown, Reason: "subscription status: pending"},
		refreshStatus: &SubscriptionStatus{OrganizationID: 42, IsActive: false, Status: StatusUnknown, Reason: "subscription status: pending"},
	}
	mw := NewMiddleware(provider, nil)

	w, _ := doRequest(provider, mw.RequireActiveSubscription())

	require.Equal(t, 1, provider.refreshCalls, "an unknown status must trigger a provider refresh before the 402")
	assert.Equal(t, http.StatusPaymentRequired, w.Code)
}

func TestUnknownStatusRefreshToActiveHeals(t *testing.T) {
	provider := &fakeProvider{
		getStatus: &SubscriptionStatus{OrganizationID: 42, IsActive: false, Status: StatusUnknown, Reason: "subscription status: pending"},
		// Provider says the subscription is actually active: the webhook was missed.
		refreshStatus: &SubscriptionStatus{OrganizationID: 42, IsActive: true, Status: StatusActive, Reason: "ok"},
	}
	mw := NewMiddleware(provider, nil)

	w, captured := doRequest(provider, mw.RequireActiveSubscription())

	require.Equal(t, 1, provider.refreshCalls)
	assert.Equal(t, http.StatusOK, w.Code, "refresh-to-active must heal the request instead of denying")
	require.NotNil(t, captured)
	status := GetSubscriptionStatus(captured)
	require.NotNil(t, status, "healed subscription must be set in context")
	assert.True(t, status.IsActive)
}

func TestUnknownStatusDenialNeverReportsNone(t *testing.T) {
	provider := &fakeProvider{
		getStatus: &SubscriptionStatus{OrganizationID: 42, IsActive: false, Status: StatusUnknown, Reason: "subscription status: revoked"},
	}
	mw := NewMiddleware(provider, nil)

	w, _ := doRequest(provider, mw.RequireActiveSubscription())

	require.Equal(t, http.StatusPaymentRequired, w.Code)
	var body ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, StatusUnknown, body.Status, "the 402 must carry the distinct unknown status")
	assert.NotEqual(t, StatusNone, body.Status, "an existing-but-unrecognized subscription must never report status 'none'")
}

func TestNoneStatusSkipsRefresh(t *testing.T) {
	provider := &fakeProvider{
		getStatus: &SubscriptionStatus{OrganizationID: 42, IsActive: false, Status: StatusNone, Reason: "no active subscription found"},
	}
	mw := NewMiddleware(provider, nil)

	w, _ := doRequest(provider, mw.RequireActiveSubscription())

	assert.Equal(t, 0, provider.refreshCalls, "no subscription (status none) must skip the provider refresh")
	assert.Equal(t, http.StatusPaymentRequired, w.Code)
}

// === Boundary tests (new-client-billing-lifecycle, task 3.3) ===

func TestErrNoSubscriptionReturns402SubscriptionRequired(t *testing.T) {
	// Design D1/D4: the adapter translates ErrSubscriptionNotFound →
	// ErrNoSubscription; the middleware classifies it as 402 subscription_required
	// (status none), never a generic 402 or a 500.
	provider := &fakeProvider{
		getErr: ErrNoSubscription,
	}
	mw := NewMiddleware(provider, nil)

	w, _ := doRequest(provider, mw.RequireActiveSubscription())

	assert.Equal(t, http.StatusPaymentRequired, w.Code)
	assert.Equal(t, 0, provider.refreshCalls, "no-subscription must not trigger a provider refresh")

	var body ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "subscription_required", body.Error, "no-subscription must carry subscription_required")
	assert.Equal(t, StatusNone, body.Status)
	assert.Contains(t, body.UpgradeURL, "dashboard/settings")
}

func TestDBErrorReturns500Not402(t *testing.T) {
	// Design D1: any error other than ErrNoSubscription (e.g. DB failure)
	// must return HTTP 500, never a misleading 402.
	provider := &fakeProvider{
		getErr: assert.AnError, // generic error
	}
	mw := NewMiddleware(provider, nil)

	w, _ := doRequest(provider, mw.RequireActiveSubscription())

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var body ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "subscription_check_failed", body.Error)
}

func TestTrialExpiredBoundary_LazyGuardFires(t *testing.T) {
	// Design D6: a trial-expired org has a real subscription row (status none
	// would skip the guard), but the DB says inactive. The lazy guard fires,
	// and when the provider confirms expiry, the 402 is returned.
	provider := &fakeProvider{
		getStatus:     &SubscriptionStatus{OrganizationID: 42, IsActive: false, Status: StatusTrialing, Reason: "trial expired"},
		refreshStatus: &SubscriptionStatus{OrganizationID: 42, IsActive: false, Status: StatusTrialing, Reason: "trial expired"},
	}
	mw := NewMiddleware(provider, nil)

	w, _ := doRequest(provider, mw.RequireActiveSubscription())

	// The guard fires because IsActive=false and Status != StatusNone.
	assert.Equal(t, 1, provider.refreshCalls, "trial-expired (not 'none') must trigger the lazy guard")
	assert.Equal(t, http.StatusPaymentRequired, w.Code)
}

func TestTrialExpiredBoundary_ProviderHealsToActive(t *testing.T) {
	// Design D6: a provider-backed trial that auto-converted (webhook missed)
	// heals via the lazy guard and access is granted.
	provider := &fakeProvider{
		getStatus:     &SubscriptionStatus{OrganizationID: 42, IsActive: false, Status: StatusTrialing, Reason: "trial expired"},
		refreshStatus: &SubscriptionStatus{OrganizationID: 42, IsActive: true, Status: StatusActive, Reason: "ok"},
	}
	mw := NewMiddleware(provider, nil)

	w, captured := doRequest(provider, mw.RequireActiveSubscription())

	assert.Equal(t, 1, provider.refreshCalls)
	assert.Equal(t, http.StatusOK, w.Code, "provider-refresh-to-active must grant access")
	require.NotNil(t, captured)
	status := GetSubscriptionStatus(captured)
	require.NotNil(t, status)
	assert.True(t, status.IsActive)
}

func TestUpgradeURLDefaultPointsToSettingsSubscription(t *testing.T) {
	cfg := DefaultMiddlewareConfig()
	assert.Equal(t, "/dashboard/settings?view=subscription", cfg.UpgradeURL,
		"default upgrade URL must be the real settings subscription tab")
}
