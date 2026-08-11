package cognitive

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	billingServices "github.com/moasq/go-b2b-starter/internal/modules/billing/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// fakeBillingService implements billingServices.BillingService for guard tests.
type fakeBillingService struct {
	billingServices.BillingService
	status *domain.AiUsageStatus
	err    error
}

func (f *fakeBillingService) GetAiUsageStatus(ctx context.Context, organizationID int32) (*domain.AiUsageStatus, error) {
	return f.status, f.err
}

func newTestContext(orgID int32) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/example_cognitive/chat", nil)
	c.Set("organization_id", orgID)
	// Simulate the org_context middleware having run (authcontext.GetOrganizationID
	// reads the RequestContext, not the raw key).
	authcontext.SetRequestContext(c, &authcontext.RequestContext{OrganizationID: orgID})
	// Simulate the entitlement middleware having run (needed by features.Require)
	features.SetEntitlement(c, &features.Entitlement{
		Features: map[string]bool{"ai_assistant": true},
	})
	return c, rec
}

func TestAiCreditGuard_AllowsWhenCreditsRemain(t *testing.T) {
	guard := NewAiCreditGuard(&fakeBillingService{status: &domain.AiUsageStatus{
		OrganizationID:   1,
		CreditsMax:       1000,
		CreditsRemaining: 250,
	}})

	c, rec := newTestContext(1)
	called := false
	guard.RequireCredits()(c)
	if !c.IsAborted() {
		called = true
	}
	c.Next()

	assert.False(t, c.IsAborted())
	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAiCreditGuard_BlocksWhenExhausted(t *testing.T) {
	guard := NewAiCreditGuard(&fakeBillingService{status: &domain.AiUsageStatus{
		OrganizationID:   1,
		CreditsMax:       1000,
		CreditsRemaining: 0,
	}})

	c, rec := newTestContext(1)
	guard.RequireCredits()(c)

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusPaymentRequired, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "ai_credits_exhausted", body["error"])
}

func TestAiCreditGuard_AllowsWhenNoAllowanceConfigured(t *testing.T) {
	guard := NewAiCreditGuard(&fakeBillingService{status: &domain.AiUsageStatus{
		OrganizationID:   1,
		CreditsMax:       0,
		CreditsRemaining: 0,
	}})

	c, rec := newTestContext(1)
	guard.RequireCredits()(c)

	assert.False(t, c.IsAborted())
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAiCreditGuard_FailsOpenOnLedgerError(t *testing.T) {
	guard := NewAiCreditGuard(&fakeBillingService{err: errors.New("db down")})

	c, rec := newTestContext(1)
	guard.RequireCredits()(c)

	assert.False(t, c.IsAborted())
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAiCreditGuard_MissingOrgContextBlocks(t *testing.T) {
	guard := NewAiCreditGuard(&fakeBillingService{})

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/example_cognitive/chat", nil)
	// No organization_id set
	guard.RequireCredits()(c)

	assert.True(t, c.IsAborted())
}

func TestFeaturesRequireAIAssistant_Disabled(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/example_cognitive/chat", nil)
	c.Set("organization_id", 1)
	features.SetEntitlement(c, &features.Entitlement{
		Features: map[string]bool{"ai_assistant": false},
	})

	features.Require("ai_assistant")(c)

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestFeaturesRequireAIAssistant_Enabled(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/example_cognitive/chat", nil)
	c.Set("organization_id", 1)
	features.SetEntitlement(c, &features.Entitlement{
		Features: map[string]bool{"ai_assistant": true},
	})

	features.Require("ai_assistant")(c)

	assert.False(t, c.IsAborted())
	assert.Equal(t, http.StatusOK, rec.Code)
}
