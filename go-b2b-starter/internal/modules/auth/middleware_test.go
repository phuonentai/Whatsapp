package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// stubProvider is a minimal AuthProvider used to prove that RequireAuth
// performs independent token verification when X-Forwarded-Auth is absent,
// regardless of any forwarded org/member headers the client supplied.
type stubProvider struct {
	verifyCalled bool
	identity     *Identity
	err          error
}

func (p *stubProvider) VerifyToken(ctx context.Context, token string) (*Identity, error) {
	p.verifyCalled = true
	if p.err != nil {
		return nil, p.err
	}
	return p.identity, nil
}

func TestRequireAuth_EdgeForwardedAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		headers        map[string]string
		expectedStatus int
	}{
		{
			name: "valid edge auth headers pass through",
			headers: map[string]string{
				"X-Forwarded-Auth":         "true",
				"X-Stytch-Organization-Id": "550e8400-e29b-41d4-a716-446655440000",
				"X-Stytch-Member-Id":       "660e8400-e29b-41d4-a716-446655440001",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing X-Forwarded-Auth is ignored (no auth header)",
			headers: map[string]string{
				"X-Stytch-Organization-Id": "550e8400-e29b-41d4-a716-446655440000",
				"X-Stytch-Member-Id":       "660e8400-e29b-41d4-a716-446655440001",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "missing org id returns 401",
			headers: map[string]string{
				"X-Forwarded-Auth":   "true",
				"X-Stytch-Member-Id": "660e8400-e29b-41d4-a716-446655440001",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "missing member id returns 401",
			headers: map[string]string{
				"X-Forwarded-Auth":         "true",
				"X-Stytch-Organization-Id": "550e8400-e29b-41d4-a716-446655440000",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "malformed org id UUID returns 401",
			headers: map[string]string{
				"X-Forwarded-Auth":         "true",
				"X-Stytch-Organization-Id": "not-a-uuid",
				"X-Stytch-Member-Id":       "660e8400-e29b-41d4-a716-446655440001",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "malformed member id UUID returns 401",
			headers: map[string]string{
				"X-Forwarded-Auth":         "true",
				"X-Stytch-Organization-Id": "550e8400-e29b-41d4-a716-446655440000",
				"X-Stytch-Member-Id":       "bad-uuid",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "X-Forwarded-Auth false is ignored",
			headers: map[string]string{
				"X-Forwarded-Auth":         "false",
				"X-Stytch-Organization-Id": "550e8400-e29b-41d4-a716-446655440000",
				"X-Stytch-Member-Id":       "660e8400-e29b-41d4-a716-446655440001",
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Middleware{
				config: &MiddlewareConfig{
					ErrorHandler:       defaultErrorHandler,
					TrustForwardedAuth: true,
				},
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

			for k, v := range tt.headers {
				c.Request.Header.Set(k, v)
			}

			m.RequireAuth()(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestRequireAuth_IdentitySetOnValidEdgeAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	m := &Middleware{
		config: &MiddlewareConfig{
			ErrorHandler:       defaultErrorHandler,
			TrustForwardedAuth: true,
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set("X-Forwarded-Auth", "true")
	c.Request.Header.Set("X-Stytch-Organization-Id", "550e8400-e29b-41d4-a716-446655440000")
	c.Request.Header.Set("X-Stytch-Member-Id", "660e8400-e29b-41d4-a716-446655440001")

	handlerCalled := false
	m.RequireAuth()(c)
	if !c.IsAborted() {
		handlerCalled = true
	}

	if !handlerCalled {
		t.Fatal("handler was not called, request was aborted")
	}

	identity := GetIdentity(c)
	if identity == nil {
		t.Fatal("expected Identity to be set in context")
	}

	if identity.UserID != "660e8400-e29b-41d4-a716-446655440001" {
		t.Errorf("expected UserID %q, got %q", "660e8400-e29b-41d4-a716-446655440001", identity.UserID)
	}

	if identity.OrganizationID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("expected OrganizationID %q, got %q", "550e8400-e29b-41d4-a716-446655440000", identity.OrganizationID)
	}
}

// TestRequireAuth_NoForwardedAuthHeaderUsesTokenVerification asserts that the
// forwarded org/member headers are never trusted on their own: without
// X-Forwarded-Auth the middleware must verify the Bearer token independently
// (even with TrustForwardedAuth enabled), and must NOT derive the identity
// from user-supplied X-Stytch-* headers.
func TestRequireAuth_NoForwardedAuthHeaderUsesTokenVerification(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokenIdentity := &Identity{
		UserID:         "token-member-id",
		OrganizationID: "token-org-id",
		Email:          "member@test.com",
		EmailVerified:  true,
	}

	provider := &stubProvider{identity: tokenIdentity}
	m := &Middleware{
		provider: provider,
		config: &MiddlewareConfig{
			ErrorHandler:       defaultErrorHandler,
			TrustForwardedAuth: true, // fast path is enabled, but headers alone must not be trusted
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer valid.jwt.token")
	// Client-supplied forwarded identity headers that must be IGNORED because
	// X-Forwarded-Auth is absent.
	c.Request.Header.Set("X-Stytch-Organization-Id", "550e8400-e29b-41d4-a716-446655440000")
	c.Request.Header.Set("X-Stytch-Member-Id", "660e8400-e29b-41d4-a716-446655440001")

	handlerCalled := false
	handler := m.RequireAuth()
	handler(c)
	if !c.IsAborted() {
		handlerCalled = true
	}

	if !provider.verifyCalled {
		t.Fatal("expected provider.VerifyToken to be called (independent token verification)")
	}
	if !handlerCalled {
		t.Fatal("handler was not called, request was aborted despite a valid token")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	identity := GetIdentity(c)
	if identity == nil {
		t.Fatal("expected Identity to be set in context")
	}
	if identity.UserID != "token-member-id" {
		t.Errorf("expected identity from token UserID %q, got %q (forwarded header must not be trusted)", "token-member-id", identity.UserID)
	}
	if identity.OrganizationID != "token-org-id" {
		t.Errorf("expected identity from token OrganizationID %q, got %q (forwarded header must not be trusted)", "token-org-id", identity.OrganizationID)
	}
}

// TestRequireAuth_NoForwardedAuthHeaderInvalidTokenRejected asserts that when
// X-Forwarded-Auth is absent and the Bearer token fails verification, the
// request is rejected even though valid-looking forwarded headers are present.
func TestRequireAuth_NoForwardedAuthHeaderInvalidTokenRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)

	provider := &stubProvider{err: ErrInvalidToken}
	m := &Middleware{
		provider: provider,
		config: &MiddlewareConfig{
			ErrorHandler:       defaultErrorHandler,
			TrustForwardedAuth: true,
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer invalid.jwt.token")
	c.Request.Header.Set("X-Stytch-Organization-Id", "550e8400-e29b-41d4-a716-446655440000")
	c.Request.Header.Set("X-Stytch-Member-Id", "660e8400-e29b-41d4-a716-446655440001")

	m.RequireAuth()(c)

	if !provider.verifyCalled {
		t.Fatal("expected provider.VerifyToken to be called (independent token verification)")
	}
	if !c.IsAborted() {
		t.Fatal("expected request to be aborted for an invalid token")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}
