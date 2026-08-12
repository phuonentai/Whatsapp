package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// stubResolver serves the named middleware registered in the map, defaulting to a no-op.
type stubResolver struct {
	middlewares map[string]gin.HandlerFunc
}

func (s *stubResolver) Get(name string) gin.HandlerFunc {
	if h, ok := s.middlewares[name]; ok {
		return h
	}
	return func(c *gin.Context) { c.Next() }
}

// newRBACTestRouter builds a router with the real Routes registration and the real
// auth middleware. Mock auth (X-Test-Org-ID bypass) is enabled only for the
// authenticated scenarios.
func newRBACTestRouter(enableMockAuth bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api")

	config := DefaultMiddlewareConfig()
	config.EnableMockAuth = enableMockAuth
	mw := NewMiddleware(nil, nil, nil, config)
	resolver := &stubResolver{middlewares: map[string]gin.HandlerFunc{
		"auth": mw.RequireAuth(),
	}}

	routes := NewRoutes(NewHandler(NewRBACService()))
	routes.RegisterRoutes(api, resolver)
	return router
}

func TestRBACRoutes_RequireAuth(t *testing.T) {
	cases := []struct {
		name        string
		method      string
		path        string
		body        string
		responseKey string
	}{
		{"roles", http.MethodGet, "/api/rbac/roles", "", "roles"},
		{"permissions", http.MethodGet, "/api/rbac/permissions", "", "permissions"},
		{"permissions-by-category", http.MethodGet, "/api/rbac/permissions/by-category", "", "categories"},
		{"role-details", http.MethodGet, "/api/rbac/roles/admin", "", "role"},
		{"check-permission", http.MethodPost, "/api/rbac/check-permission", `{"role_id":"admin","permission_id":"org:manage"}`, "role_id"},
		{"metadata", http.MethodGet, "/api/rbac/metadata", "", "total_roles"},
	}

	t.Run("unauthenticated returns 401 on all six endpoints", func(t *testing.T) {
		router := newRBACTestRouter(false)
		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				if w.Code != http.StatusUnauthorized {
					t.Errorf("%s %s: got status %d, want 401 (body: %s)", tt.method, tt.path, w.Code, w.Body.String())
				}
			})
		}
	})

	t.Run("authenticated returns 200 with unchanged JSON shape", func(t *testing.T) {
		router := newRBACTestRouter(true)
		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
				req.Header.Set("X-Test-Org-ID", "test-org-pro:admin-pro@test.com")
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				if w.Code != http.StatusOK {
					t.Fatalf("%s %s: got status %d, want 200 (body: %s)", tt.method, tt.path, w.Code, w.Body.String())
				}
				var payload map[string]any
				if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
					t.Fatalf("%s %s: response is not JSON: %v", tt.method, tt.path, err)
				}
				// Handlers wrap responses in a {"data": ..., "success": true} envelope.
				data, ok := payload["data"]
				if !ok {
					t.Fatalf("%s %s: response missing envelope key %q (body: %s)", tt.method, tt.path, "data", w.Body.String())
				}
				dataObj, ok := data.(map[string]any)
				if !ok {
					t.Fatalf("%s %s: envelope data is not an object (body: %s)", tt.method, tt.path, w.Body.String())
				}
				if _, ok := dataObj[tt.responseKey]; !ok {
					t.Errorf("%s %s: response data missing key %q (body: %s)", tt.method, tt.path, tt.responseKey, w.Body.String())
				}
			})
		}
	})
}
