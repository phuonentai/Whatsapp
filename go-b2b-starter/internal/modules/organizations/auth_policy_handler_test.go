package organizations

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/internal/modules/organizations/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/organizations/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
)

// stubAuthPolicyService wraps the real OrganizationService interface and
// stubs the auth-policy methods; every other method stays unreachable in
// these tests (the embedded interface keeps the handler signature).
type stubAuthPolicyService struct {
	services.OrganizationService
	getErr       error
	updateErr    error
	policy       *domain.AuthPolicy
	updateCalls  int
	lastEmailJIT domain.JitPolicy
}

func (s *stubAuthPolicyService) GetAuthPolicy(ctx context.Context, orgID string) (*domain.AuthPolicy, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.policy == nil {
		return &domain.AuthPolicy{
			EmailJITProvisioning: domain.JitPolicyDisabled,
			SSOJITProvisioning:   domain.SsoJitPolicyDisabled,
		}, nil
	}
	return s.policy, nil
}

func (s *stubAuthPolicyService) UpdateAuthPolicy(
	ctx context.Context,
	orgID string,
	emailJitPolicy domain.JitPolicy,
	allowedDomains []string,
	allowedAuthMethods []domain.AllowedAuthMethod,
	ssoJitPolicy domain.SsoJitPolicy,
	ssoJitAllowedConnections []string,
	ssoDefaultConnectionID string,
) error {
	s.updateCalls++
	s.lastEmailJIT = emailJitPolicy
	if s.updateErr != nil {
		return s.updateErr
	}
	if orgID == "" {
		return domain.ErrAuthOrganizationIDRequired
	}
	return nil
}

// newAuthPolicyTestRouter wires the real routes (auth + org:manage + the
// auth-policy handlers) with mock auth. Mock identity comes from the
// X-Test-Org-ID header ("<orgID>:<email>").
func newAuthPolicyTestRouter(svc *stubAuthPolicyService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api")

	config := auth.DefaultMiddlewareConfig()
	config.EnableMockAuth = true
	mw := auth.NewMiddleware(nil, nil, nil, config)

	resolver := &stubResolver{middlewares: map[string]gin.HandlerFunc{
		"auth": mw.RequireAuth(),
		"org_context": func(c *gin.Context) {
			identity := authcontext.GetIdentity(c)
			if identity == nil {
				c.Next()
				return
			}
			authcontext.SetRequestContext(c, &authcontext.RequestContext{
				Identity:       identity,
				OrganizationID: 1,
				AccountID:      1,
				ProviderOrgID:  identity.OrganizationID,
			})
			c.Next()
		},
	}}

	handler := NewOrganizationHandler(svc, noopLogger{})
	orgGroup := api.Group("/organizations")
	orgGroup.Use(
		resolver.Get("auth"),
		resolver.Get("org_context"),
	)
	orgGroup.GET("/auth-policy",
		auth.RequirePermissionFunc("org", "manage"),
		handler.GetAuthPolicy)
	orgGroup.PUT("/auth-policy",
		auth.RequirePermissionFunc("org", "manage"),
		handler.UpdateAuthPolicy)

	return router
}

func doAuthPolicyGet(router *gin.Engine, testOrgHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/organizations/auth-policy", nil)
	if testOrgHeader != "" {
		req.Header.Set("X-Test-Org-ID", testOrgHeader)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func doAuthPolicyPut(router *gin.Engine, testOrgHeader string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPut, "/api/organizations/auth-policy", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if testOrgHeader != "" {
		req.Header.Set("X-Test-Org-ID", testOrgHeader)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

const validAuthPolicyBody = `{
	"email_jit_provisioning":"DISABLED",
	"email_allowed_domains":null,
	"allowed_auth_methods":["magic_link","email_otp"],
	"sso_jit_provisioning":"DISABLED",
	"sso_jit_provisioning_allowed_connections":null,
	"sso_default_connection_id":""
}`

func TestGetAuthPolicyHandlerSuccess(t *testing.T) {
	svc := &stubAuthPolicyService{policy: &domain.AuthPolicy{
		EmailJITProvisioning:   domain.JitPolicyDomainRestricted,
		EmailAllowedDomains:    []string{"acme.com"},
		AuthMethodsRestricted:  true,
		AllowedAuthMethods:     []domain.AllowedAuthMethod{domain.AuthMethodMagicLink, domain.AuthMethodEmailOTP},
		SSOJITProvisioning:     domain.SsoJitPolicyDisabled,
		SSOActiveConnectionIDs: []string{"conn-1"},
	}}
	router := newAuthPolicyTestRouter(svc)

	w := doAuthPolicyGet(router, "org-1:admin@test.com")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			EmailJITProvisioning   string   `json:"email_jit_provisioning"`
			AllowedAuthMethods     []string `json:"allowed_auth_methods"`
			SSOActiveConnectionIDs []string `json:"sso_active_connection_ids"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %s", w.Body.String())
	}
	if resp.Data.EmailJITProvisioning != string(domain.JitPolicyDomainRestricted) {
		t.Fatalf("expected DOMAIN_RESTRICTED mirror, got %q", resp.Data.EmailJITProvisioning)
	}
	if len(resp.Data.AllowedAuthMethods) != 2 {
		t.Fatalf("expected 2 allowed auth methods in mirror, got %v", resp.Data.AllowedAuthMethods)
	}
	if len(resp.Data.SSOActiveConnectionIDs) != 1 {
		t.Fatalf("expected 1 active connection id in mirror, got %v", resp.Data.SSOActiveConnectionIDs)
	}
}

func TestUpdateAuthPolicyHandlerSuccess(t *testing.T) {
	svc := &stubAuthPolicyService{}
	router := newAuthPolicyTestRouter(svc)

	w := doAuthPolicyPut(router, "org-1:admin@test.com", validAuthPolicyBody)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	if svc.updateCalls != 1 {
		t.Fatalf("expected 1 service call, got %d", svc.updateCalls)
	}
	if svc.lastEmailJIT != domain.JitPolicyDisabled {
		t.Fatalf("expected DISABLED email JIT, got %q", svc.lastEmailJIT)
	}
}

func TestAuthPolicyHandlersRequireOrgManage(t *testing.T) {
	svc := &stubAuthPolicyService{}
	router := newAuthPolicyTestRouter(svc)

	if w := doAuthPolicyGet(router, "org-1:member-jane@test.com"); w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 on GET without org:manage, got %d (body: %s)", w.Code, w.Body.String())
	}
	if w := doAuthPolicyPut(router, "org-1:member-jane@test.com", validAuthPolicyBody); w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 on PUT without org:manage, got %d (body: %s)", w.Code, w.Body.String())
	}
	if svc.updateCalls != 0 {
		t.Fatalf("expected no service call without permission, got %d", svc.updateCalls)
	}
}

func TestAuthPolicyHandlersUnauthenticated(t *testing.T) {
	svc := &stubAuthPolicyService{}
	router := newAuthPolicyTestRouter(svc)

	if w := doAuthPolicyGet(router, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on GET without auth, got %d", w.Code)
	}
	if w := doAuthPolicyPut(router, "", validAuthPolicyBody); w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on PUT without auth, got %d", w.Code)
	}
}

func TestUpdateAuthPolicyHandlerInvalidPayload(t *testing.T) {
	svc := &stubAuthPolicyService{}
	router := newAuthPolicyTestRouter(svc)

	// Invalid email JIT enum value: the service rejects it and the handler
	// maps the invalid-auth-policy sentinel to 400.
	svc.updateErr = domain.ErrInvalidAuthPolicy
	w := doAuthPolicyPut(router, "org-1:admin@test.com", `{"email_jit_provisioning":"MAYBE"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid policy, got %d (body: %s)", w.Code, w.Body.String())
	}
	if svc.updateCalls != 1 {
		t.Fatalf("expected 1 service call that rejects the payload, got %d", svc.updateCalls)
	}
}

func TestGetAuthPolicyHandlerBreakerOpenReturns503(t *testing.T) {
	svc := &stubAuthPolicyService{getErr: domain.ErrAuthPolicyUnavailable}
	router := newAuthPolicyTestRouter(svc)

	w := doAuthPolicyGet(router, "org-1:admin@test.com")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d (body: %s)", w.Code, w.Body.String())
	}

	var resp struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != "auth_policy_unavailable" {
		t.Fatalf("expected structured code auth_policy_unavailable, got %q", resp.Code)
	}
}

func TestUpdateAuthPolicyHandlerBreakerOpenReturns503(t *testing.T) {
	svc := &stubAuthPolicyService{updateErr: domain.ErrAuthPolicyUnavailable}
	router := newAuthPolicyTestRouter(svc)

	w := doAuthPolicyPut(router, "org-1:admin@test.com", validAuthPolicyBody)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d (body: %s)", w.Code, w.Body.String())
	}

	var resp struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != "auth_policy_update_unavailable" {
		t.Fatalf("expected structured code auth_policy_update_unavailable, got %q", resp.Code)
	}
}
