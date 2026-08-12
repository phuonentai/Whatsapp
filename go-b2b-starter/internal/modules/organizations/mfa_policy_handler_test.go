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
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
)

type noopLogger struct{}

func (noopLogger) Debug(msg string, fields ...logger.Fields) {}
func (noopLogger) Info(msg string, fields ...logger.Fields)  {}
func (noopLogger) Warn(msg string, fields ...logger.Fields)  {}
func (noopLogger) Error(msg string, fields ...logger.Fields) {}
func (noopLogger) Fatal(msg string, fields ...logger.Fields) {}
func (noopLogger) WithFields(fields logger.Fields) logger.Logger {
	return noopLogger{}
}

// stubResolver serves the named middleware registered in the map, defaulting
// to a no-op (mirrors the auth module's test resolver).
type stubResolver struct {
	middlewares map[string]gin.HandlerFunc
}

func (s *stubResolver) Get(name string) gin.HandlerFunc {
	if h, ok := s.middlewares[name]; ok {
		return h
	}
	return func(c *gin.Context) { c.Next() }
}

// stubOrgService wraps the real OrganizationService interface and records
// UpdateMfaPolicy calls; the embedded interface keeps the handler signature
// while every other method stays unreachable in these tests.
type stubOrgService struct {
	services.OrganizationService
	updateErr   error
	updateCalls int
	lastOrgID   string
	lastPolicy  domain.MfaPolicy
}

func (s *stubOrgService) UpdateMfaPolicy(
	ctx context.Context,
	orgID string,
	policy domain.MfaPolicy,
	methods domain.MfaMethods,
	allowedMethods []domain.MfaMethod,
) error {
	s.updateCalls++
	s.lastOrgID = orgID
	s.lastPolicy = policy
	if s.updateErr != nil {
		return s.updateErr
	}
	// Mirror the real service's validation so handler tests exercise the same
	// 400 boundary without needing the full repository stack.
	if orgID == "" {
		return domain.ErrAuthOrganizationIDRequired
	}
	return domain.ValidateMfaPolicyUpdate(policy, methods, allowedMethods)
}

// newMfaPolicyTestRouter wires the real routes (auth + org:manage) with mock
// auth. Mock identity comes from the X-Test-Org-ID header
// ("<orgID>:<email>"); the org_context middleware is stubbed to a fixed
// RequestContext carrying the same provider org ID.
func newMfaPolicyTestRouter(svc *stubOrgService) *gin.Engine {
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
	orgGroup.PUT("/mfa-policy",
		auth.RequirePermissionFunc("org", "manage"),
		handler.UpdateMfaPolicy)

	return router
}

func doMfaPolicyPut(router *gin.Engine, testOrgHeader string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPut, "/api/organizations/mfa-policy", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if testOrgHeader != "" {
		req.Header.Set("X-Test-Org-ID", testOrgHeader)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

const validPolicyBody = `{"mfa_policy":"REQUIRED_FOR_ALL","mfa_methods":"RESTRICTED","allowed_mfa_methods":["totp"]}`

func TestUpdateMfaPolicyHandlerSuccess(t *testing.T) {
	svc := &stubOrgService{}
	router := newMfaPolicyTestRouter(svc)

	w := doMfaPolicyPut(router, "org-1:admin@test.com", validPolicyBody)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Updated bool `json:"updated"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Success || !resp.Data.Updated {
		t.Fatalf("expected success/updated=true, got %s", w.Body.String())
	}
	if svc.updateCalls != 1 {
		t.Fatalf("expected 1 service call, got %d", svc.updateCalls)
	}
	if svc.lastOrgID != "org-1" {
		t.Fatalf("expected provider org id org-1, got %q", svc.lastOrgID)
	}
	if svc.lastPolicy != domain.MfaPolicyRequiredForAll {
		t.Fatalf("expected REQUIRED_FOR_ALL policy, got %q", svc.lastPolicy)
	}
}

func TestUpdateMfaPolicyHandlerRequiresOrgManage(t *testing.T) {
	svc := &stubOrgService{}
	router := newMfaPolicyTestRouter(svc)

	// Member mock identity ("<role>-<name>@test.com"): mockPermissionsForRole
	// strips org:manage for non-admin roles.
	w := doMfaPolicyPut(router, "org-1:member-jane@test.com", validPolicyBody)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without org:manage, got %d (body: %s)", w.Code, w.Body.String())
	}
	if svc.updateCalls != 0 {
		t.Fatalf("expected no service call without permission, got %d", svc.updateCalls)
	}
}

func TestUpdateMfaPolicyHandlerUnauthenticated(t *testing.T) {
	svc := &stubOrgService{}
	router := newMfaPolicyTestRouter(svc)

	w := doMfaPolicyPut(router, "", validPolicyBody)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", w.Code)
	}
}

func TestUpdateMfaPolicyHandlerInvalidPayload(t *testing.T) {
	svc := &stubOrgService{}
	router := newMfaPolicyTestRouter(svc)

	w := doMfaPolicyPut(router, "org-1:admin@test.com", `{"mfa_policy":"NEVER","mfa_methods":"ALL_ALLOWED"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid policy, got %d (body: %s)", w.Code, w.Body.String())
	}
	// Validation lives in the service: the handler delegates and maps the
	// rejected payload to 400.
	if svc.updateCalls != 1 {
		t.Fatalf("expected 1 service call that rejects the payload, got %d", svc.updateCalls)
	}
}

func TestUpdateMfaPolicyHandlerBreakerOpenReturns503(t *testing.T) {
	svc := &stubOrgService{updateErr: domain.ErrMfaPolicyUnavailable}
	router := newMfaPolicyTestRouter(svc)

	w := doMfaPolicyPut(router, "org-1:admin@test.com", validPolicyBody)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d (body: %s)", w.Code, w.Body.String())
	}

	var resp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != "mfa_policy_update_unavailable" {
		t.Fatalf("expected structured code mfa_policy_update_unavailable, got %q", resp.Code)
	}
}
