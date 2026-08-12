package billing

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	serverDomain "github.com/moasq/go-b2b-starter/internal/platform/server/domain"
)

// stubMiddlewareResolver returns no-op middleware for any requested name.
type stubMiddlewareResolver struct{}

func (stubMiddlewareResolver) Get(name string) gin.HandlerFunc {
	return func(c *gin.Context) { c.Next() }
}

func newRoutesEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	handler := NewHandler(nil, newWebhookVerifier(), "polar-secret", "mp-secret", noopHandlerLogger{})
	handler.Routes(engine.Group(serverDomain.ApiPrefix), stubMiddlewareResolver{})
	return engine
}

func TestRoutes_WebhookPathsResolveUnderSingleAPIPrefix(t *testing.T) {
	engine := newRoutesEngine(t)

	// The billing module mounts under /api; the webhook group is /v1/webhooks
	// (the whatsapp pattern), so the effective paths are /api/v1/webhooks/*.
	for _, path := range []string{"/api/v1/webhooks/polar", "/api/v1/webhooks/mercadopago"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
		engine.ServeHTTP(w, req)
		// Signature verification rejects the unsigned request (401) — the
		// important assertion is that the route resolved (no 404).
		assert.NotEqual(t, http.StatusNotFound, w.Code, "POST %s must resolve", path)
		assert.Equal(t, http.StatusUnauthorized, w.Code, "unsigned webhook POST %s must fail signature check", path)
	}
}

func TestRoutes_DuplicatedAPIPrefixNoLongerResolves(t *testing.T) {
	engine := newRoutesEngine(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/api/v1/webhooks/polar", strings.NewReader(`{}`))
	engine.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code, "the duplicated /api/api path must 404")

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/api/v1/webhooks/mercadopago", strings.NewReader(`{}`))
	engine.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusNotFound, w2.Code, "the duplicated /api/api path must 404")
}

