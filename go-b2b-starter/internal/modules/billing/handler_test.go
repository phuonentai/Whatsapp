package billing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	billingServices "github.com/moasq/go-b2b-starter/internal/modules/billing/app/services"
	billingDomain "github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

type noopHandlerLogger struct{}

func (noopHandlerLogger) Debug(msg string, fields ...logger.Fields)                     {}
func (noopHandlerLogger) Info(msg string, fields ...logger.Fields)                      {}
func (noopHandlerLogger) Warn(msg string, fields ...logger.Fields)                      {}
func (noopHandlerLogger) Error(msg string, fields ...logger.Fields)                     {}
func (noopHandlerLogger) Fatal(msg string, fields ...logger.Fields)                     {}
func (noopHandlerLogger) WithFields(fields logger.Fields) logger.Logger                  { return noopHandlerLogger{} }

// stubBillingService records the explicit org parameters passed by the
// handlers; everything else is unimplemented via the embedded interface.
type stubBillingService struct {
	billingServices.BillingService
	createArgs struct{ stytchOrgID, planID string }
	cancelArgs struct{ stytchOrgID, subscriptionID string }
}

func (s *stubBillingService) CreateMPCheckout(ctx context.Context, stytchOrgID, planID string) (*billingDomain.BillingStatus, error) {
	s.createArgs.stytchOrgID = stytchOrgID
	s.createArgs.planID = planID
	return &billingDomain.BillingStatus{CheckoutURL: "https://checkout.example/x"}, nil
}

func (s *stubBillingService) CancelMPSubscription(ctx context.Context, stytchOrgID, subscriptionID string) (*billingDomain.BillingStatus, error) {
	s.cancelArgs.stytchOrgID = stytchOrgID
	s.cancelArgs.subscriptionID = subscriptionID
	return &billingDomain.BillingStatus{}, nil
}

func newTestHandler(stub *stubBillingService) *Handler {
	return NewHandler(stub, newWebhookVerifier(), "polar-secret", "mp-secret", noopHandlerLogger{})
}

func postJSON(t *testing.T, body string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c
}

func TestCreateMPCheckout_ResolvesOrgFromGinContext(t *testing.T) {
	stub := &stubBillingService{}
	h := newTestHandler(stub)

	c := postJSON(t, `{"plan_id":"plan-1"}`)
	c.Set("stytch_org_id", "org_123")
	h.CreateMPCheckout(c)

	assert.Equal(t, http.StatusOK, c.Writer.Status())
	assert.Equal(t, "org_123", stub.createArgs.stytchOrgID)
	assert.Equal(t, "plan-1", stub.createArgs.planID)
}

func TestCreateMPCheckout_MissingGinOrgIsRejected(t *testing.T) {
	stub := &stubBillingService{}
	h := newTestHandler(stub)

	c := postJSON(t, `{"plan_id":"plan-1"}`)
	h.CreateMPCheckout(c)

	assert.Equal(t, http.StatusUnauthorized, c.Writer.Status())
	assert.Empty(t, stub.createArgs.stytchOrgID)
}

func TestCancelMPSubscription_ResolvesOrgFromGinContext(t *testing.T) {
	stub := &stubBillingService{}
	h := newTestHandler(stub)

	c := postJSON(t, `{"subscription_id":"pre-1"}`)
	c.Set("stytch_org_id", "org_456")
	h.CancelMPSubscription(c)

	assert.Equal(t, http.StatusOK, c.Writer.Status())
	assert.Equal(t, "org_456", stub.cancelArgs.stytchOrgID)
	assert.Equal(t, "pre-1", stub.cancelArgs.subscriptionID)
}

func TestCancelMPSubscription_MissingGinOrgIsRejected(t *testing.T) {
	stub := &stubBillingService{}
	h := newTestHandler(stub)

	c := postJSON(t, `{"subscription_id":"pre-1"}`)
	h.CancelMPSubscription(c)

	assert.Equal(t, http.StatusUnauthorized, c.Writer.Status())
	assert.Empty(t, stub.cancelArgs.stytchOrgID)
}
