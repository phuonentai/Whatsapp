package analytics

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moasq/go-b2b-starter/internal/modules/analytics/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/analytics/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
)

type stubRepo struct {
	revenue  []domain.RevenuePoint
	inactive []domain.InactiveContact
}

func (s *stubRepo) RevenueByPeriod(ctx context.Context, orgID int32, from, to time.Time, period string) ([]domain.RevenuePoint, error) {
	return s.revenue, nil
}

func (s *stubRepo) TopCustomersByRevenue(ctx context.Context, orgID int32, limit int32) ([]domain.TopCustomer, error) {
	return []domain.TopCustomer{{Nombre: "Cliente A", MontoTotal: 250000}}, nil
}
func (s *stubRepo) FunnelByStage(ctx context.Context, orgID int32) ([]domain.FunnelEntry, error) {
	return nil, nil
}
func (s *stubRepo) DealStateCounts(ctx context.Context, orgID int32) ([]domain.FunnelEntry, error) {
	return nil, nil
}
func (s *stubRepo) InactiveContacts(ctx context.Context, orgID int32, since time.Time) ([]domain.InactiveContact, error) {
	return s.inactive, nil
}
func (s *stubRepo) DefaultPipelineStages(ctx context.Context, orgID int32) ([]string, error) {
	return []string{}, nil
}

type stubFeatureProvider struct {
	ent *features.Entitlement
}

func (p *stubFeatureProvider) GetEntitlement(ctx context.Context, orgID int32) (*features.Entitlement, error) {
	return p.ent, nil
}

func newTestRouter(ent *features.Entitlement, identity *auth.Identity) *gin.Engine {
	gin.SetMode(gin.TestMode)
	svc := services.NewSalesReportService(&stubRepo{
		revenue:  []domain.RevenuePoint{{Periodo: "2026-07-01", MontoTotal: 350000}},
		inactive: []domain.InactiveContact{{Telefono: "573001234567", Nombre: "Juan", Clasificacion: "inactivo"}},
	})
	routes := NewRoutes(NewHandler(svc), &stubFeatureProvider{ent: ent})

	r := gin.New()
	r.Use(func(c *gin.Context) {
		if identity != nil {
			auth.SetIdentity(c, identity)
		}
		auth.SetRequestContext(c, &auth.RequestContext{OrganizationID: 1, AccountID: 1})
		if ent != nil {
			features.SetEntitlement(c, ent)
		}
		c.Next()
	})
	routes.RegisterRoutes(r.Group("/api"), testResolver{})
	return r
}

type testResolver struct{}

func (testResolver) Get(name string) gin.HandlerFunc {
	return func(c *gin.Context) { c.Next() }
}

func withPerms(perms ...string) *auth.Identity {
	var ps []auth.Permission
	for _, p := range perms {
		resource, action := splitPerm(p)
		ps = append(ps, auth.NewPermission(resource, action))
	}
	return &auth.Identity{UserID: "member-1", Permissions: ps}
}

func splitPerm(p string) (string, string) {
	for i := 0; i < len(p); i++ {
		if p[i] == ':' {
			return p[:i], p[i+1:]
		}
	}
	return p, ""
}

func enabledEntitlement() *features.Entitlement {
	return &features.Entitlement{
		Features: map[string]bool{},
		Modules: map[string]features.ModuleState{
			"analytics": {Enabled: true, Features: []string{"analytics_module"}},
		},
	}
}

func TestRevenue_ModuleDisabled_Returns403(t *testing.T) {
	router := newTestRouter(&features.Entitlement{
		Features: map[string]bool{},
		Modules:  map[string]features.ModuleState{},
	}, withPerms("invoice:view"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/analytics/revenue", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "module_disabled", body["error"])
}

func TestRevenue_MissingPermission_Returns403(t *testing.T) {
	router := newTestRouter(enabledEntitlement(), withPerms("contact:view"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/analytics/revenue", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRevenue_Success(t *testing.T) {
	router := newTestRouter(enabledEntitlement(), withPerms("invoice:view"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/analytics/revenue?period=month&from=2026-07-01&to=2026-07-31", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Data []domain.RevenuePoint `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
	assert.Equal(t, "2026-07-01", body.Data[0].Periodo)
	assert.Equal(t, 350000.0, body.Data[0].MontoTotal)
}

func TestRevenue_InvalidRange_Returns400(t *testing.T) {
	router := newTestRouter(enabledEntitlement(), withPerms("invoice:view"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/analytics/revenue?from=2026-08-01&to=2026-07-01", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRevenue_InvalidPeriod_Returns400(t *testing.T) {
	router := newTestRouter(enabledEntitlement(), withPerms("invoice:view"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/analytics/revenue?period=year", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestInactiveContacts_InvalidDays_Returns400(t *testing.T) {
	router := newTestRouter(enabledEntitlement(), withPerms("contact:view"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/analytics/inactive-contacts?days=0", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/analytics/inactive-contacts?days=366", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestInactiveContacts_Success(t *testing.T) {
	router := newTestRouter(enabledEntitlement(), withPerms("contact:view"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/analytics/inactive-contacts?days=30", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestFunnel_RequiresDealPermission(t *testing.T) {
	router := newTestRouter(enabledEntitlement(), withPerms("contact:view"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/analytics/funnel", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}
