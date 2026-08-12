package procurement

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/procurement/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
)

// stubService implements ProcurementService for route tests by embedding the
// interface and overriding the endpoints under test.
type stubService struct {
	services.ProcurementService
	createSupplierErr error
	placeOrderErr     error
	createRunErr      error
	order             *domain.Order
}

func (s *stubService) CreateSupplier(ctx context.Context, orgID int32, in services.CreateSupplierInput, memberID string) (*domain.Supplier, error) {
	if s.createSupplierErr != nil {
		return nil, s.createSupplierErr
	}
	return &domain.Supplier{ID: 1, OrganizationID: orgID, NIT: in.NIT}, nil
}

func (s *stubService) CreateRun(ctx context.Context, orgID int32, memberID string, in services.CreateRunInput) (*domain.InquiryRun, error) {
	if s.createRunErr != nil {
		return nil, s.createRunErr
	}
	return &domain.InquiryRun{ID: 1, OrganizationID: orgID, Status: domain.RunDraft}, nil
}

func (s *stubService) PlaceOrder(ctx context.Context, orgID int32, memberID string, in services.PlaceOrderInput) (*domain.Order, error) {
	if s.placeOrderErr != nil {
		return nil, s.placeOrderErr
	}
	return s.order, nil
}

func (s *stubService) ListSuppliers(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.Supplier, error) {
	return []*domain.Supplier{}, nil
}

func (s *stubService) ListProducts(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.Product, error) {
	return []*domain.Product{}, nil
}

func (s *stubService) ListRuns(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.InquiryRun, error) {
	return []*domain.InquiryRun{}, nil
}

func (s *stubService) GetBoard(ctx context.Context, orgID, runID int32) (*domain.Board, error) {
	return nil, domain.ErrRunNotFound
}

func (s *stubService) SendRun(ctx context.Context, orgID, runID int32) (*domain.InquiryRun, error) {
	return nil, domain.ErrRunNotFound
}

func (s *stubService) ListRunOrders(ctx context.Context, orgID, runID int32) ([]*domain.Order, error) {
	return nil, nil
}

// stubResolver provides the named middleware the procurement routes require.
// authed=false simulates an unauthenticated request (401); perms controls the
// issued org permissions.
type stubResolver struct {
	authed  bool
	perms   []string
	noRoles bool
}

func (r *stubResolver) Get(name string) gin.HandlerFunc {
	switch name {
	case "auth":
		return func(c *gin.Context) {
			if !r.authed {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "unauthorized", "message": "authentication required"})
				return
			}
			perms := make([]authcontext.Permission, 0, len(r.perms))
			for _, p := range r.perms {
				perms = append(perms, authcontext.Permission(p))
			}
			authcontext.SetIdentity(c, &authcontext.Identity{
				UserID:      "stytch_user_1",
				Permissions: perms,
			})
			c.Next()
		}
	case "org_context":
		return func(c *gin.Context) {
			authcontext.SetRequestContext(c, &authcontext.RequestContext{
				Identity:       authcontext.GetIdentity(c),
				OrganizationID: 42,
				AccountID:      7,
				ProviderOrgID:  "org-uuid",
			})
			c.Next()
		}
	default:
		return func(c *gin.Context) { c.Next() }
	}
}

func newRouter(svc services.ProcurementService, resolver *stubResolver) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	routes := NewRoutes(NewHandler(svc))
	api := router.Group("/api")
	routes.RegisterRoutes(api, resolver)
	return router
}

func doJSON(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func fullPerms() []string {
	return []string{"org:view", "org:manage"}
}

func TestUnauthenticatedRejected(t *testing.T) {
	router := newRouter(&stubService{}, &stubResolver{authed: false, perms: fullPerms()})
	w := doJSON(t, router, http.MethodGet, "/api/procurement/suppliers", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestWriteWithoutManageDenied(t *testing.T) {
	svc := &stubService{}
	resolver := &stubResolver{authed: true, perms: []string{"org:view"}}
	resolver.noRoles = true
	router := newRouter(svc, resolver)
	w := doJSON(t, router, http.MethodPost, "/api/procurement/suppliers", `{"nit":"900111","phone":"+573001111111"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for write without org:manage, got %d", w.Code)
	}
	// reads are allowed with org:view
	w = doJSON(t, router, http.MethodGet, "/api/procurement/suppliers", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for read with org:view, got %d", w.Code)
	}
}

func TestCreateSupplierValidation400(t *testing.T) {
	svc := &stubService{}
	router := newRouter(svc, &stubResolver{authed: true, perms: fullPerms()})
	w := doJSON(t, router, http.MethodPost, "/api/procurement/suppliers", `{"nit":"","phone":""}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 with Spanish error, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("NIT")) {
		t.Fatalf("expected Spanish error mentioning NIT, got %s", w.Body.String())
	}
}

func TestCreateSupplierDuplicateNitMaps400(t *testing.T) {
	svc := &stubService{createSupplierErr: domain.ErrSupplierAlreadyExists}
	router := newRouter(svc, &stubResolver{authed: true, perms: fullPerms()})
	w := doJSON(t, router, http.MethodPost, "/api/procurement/suppliers", `{"nit":"900111","phone":"+573001111111"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for duplicate NIT, got %d", w.Code)
	}
}

func TestCreateRunValidation400(t *testing.T) {
	svc := &stubService{}
	router := newRouter(svc, &stubResolver{authed: true, perms: fullPerms()})
	w := doJSON(t, router, http.MethodPost, "/api/procurement/runs", `{"supplier_ids":[],"products":[]}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty run, got %d", w.Code)
	}
}

func TestPlaceOrderGuardrailChecks(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"requires human -> 400", domain.ErrResponseRequiresHuman, http.StatusBadRequest},
		{"not answered -> 400", domain.ErrResponseNotAnswered, http.StatusBadRequest},
		{"invalid run status -> 400", domain.ErrInvalidRunStatus, http.StatusBadRequest},
		{"default pipeline missing -> 400", domain.ErrDefaultPipelineMissing, http.StatusBadRequest},
		{"run not found -> 404", domain.ErrRunNotFound, http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubService{placeOrderErr: tc.err}
			router := newRouter(svc, &stubResolver{authed: true, perms: fullPerms()})
			w := doJSON(t, router, http.MethodPost, "/api/procurement/runs/1/orders",
				`{"supplier_id":1,"items":[{"product_id":10,"quantity":2}]}`)
			if w.Code != tc.want {
				t.Fatalf("expected %d, got %d (body %s)", tc.want, w.Code, w.Body.String())
			}
		})
	}
}

func TestPlaceOrderSuccess(t *testing.T) {
	svc := &stubService{order: &domain.Order{ID: 9, OrganizationID: 42, RunID: 1, SupplierID: 1, Status: domain.OrderPlaced}}
	router := newRouter(svc, &stubResolver{authed: true, perms: fullPerms()})
	w := doJSON(t, router, http.MethodPost, "/api/procurement/runs/1/orders",
		`{"supplier_id":1,"items":[{"product_id":10,"quantity":2}]}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body %s)", w.Code, w.Body.String())
	}
	var payload struct {
		Data struct {
			ID int32 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.ID != 9 {
		t.Fatalf("expected order id 9, got %d", payload.Data.ID)
	}
}

var _ = errors.New
