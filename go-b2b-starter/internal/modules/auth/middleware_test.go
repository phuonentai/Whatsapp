package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

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
				"X-Forwarded-Auth":        "true",
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
				"X-Forwarded-Auth":  "true",
				"X-Stytch-Member-Id": "660e8400-e29b-41d4-a716-446655440001",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "missing member id returns 401",
			headers: map[string]string{
				"X-Forwarded-Auth":        "true",
				"X-Stytch-Organization-Id": "550e8400-e29b-41d4-a716-446655440000",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "malformed org id UUID returns 401",
			headers: map[string]string{
				"X-Forwarded-Auth":        "true",
				"X-Stytch-Organization-Id": "not-a-uuid",
				"X-Stytch-Member-Id":       "660e8400-e29b-41d4-a716-446655440001",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "malformed member id UUID returns 401",
			headers: map[string]string{
				"X-Forwarded-Auth":        "true",
				"X-Stytch-Organization-Id": "550e8400-e29b-41d4-a716-446655440000",
				"X-Stytch-Member-Id":       "bad-uuid",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "X-Forwarded-Auth false is ignored",
			headers: map[string]string{
				"X-Forwarded-Auth":        "false",
				"X-Stytch-Organization-Id": "550e8400-e29b-41d4-a716-446655440000",
				"X-Stytch-Member-Id":       "660e8400-e29b-41d4-a716-446655440001",
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Middleware{
				config: DefaultMiddlewareConfig(),
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
		config: DefaultMiddlewareConfig(),
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
