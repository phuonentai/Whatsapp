package invoicing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
)

type stubNumerationSvc struct {
	info  *domain.NumerationInfo
	snap  *domain.NumerationSnapshot
	err   error
}

func (s *stubNumerationSvc) GetLive(ctx context.Context, orgID int32) (*domain.NumerationInfo, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.info, nil
}

func (s *stubNumerationSvc) Confirm(ctx context.Context, orgID int32) (*domain.NumerationSnapshot, error) {
	if s.err != nil {
		return nil, s.err
	}
	now := time.Now()
	return &domain.NumerationSnapshot{OrganizationID: orgID, Mode: domain.NumerationAuto, ConfirmedAt: &now}, nil
}

type stubImportSvc struct {
	counts *services.ImportCounts
	err    error
}

func (s *stubImportSvc) Preview(ctx context.Context, orgID int32) (*services.ImportCounts, error) {
	return s.counts, s.err
}

func (s *stubImportSvc) Confirm(ctx context.Context, orgID int32) (*services.ImportCounts, error) {
	return s.counts, s.err
}

func (s *stubImportSvc) DeltaSync(ctx context.Context, orgID int32) (*services.ImportCounts, error) {
	return s.counts, s.err
}

type stubTestInvoiceSvc struct {
	inv *domain.Invoice
	err error
}

func (s *stubTestInvoiceSvc) CreateTestInvoice(ctx context.Context, orgID int32) (*domain.Invoice, error) {
	return s.inv, s.err
}

type onboardingDataTestHandler struct {
	h *Handler
	r *gin.Engine
}

func newOnboardingDataTestHandler(numeration services.NumerationService, importSvc services.ImportService, testSvc services.TestInvoiceService, sandbox bool) *onboardingDataTestHandler {
	h := &Handler{
		invoicingService: &stubInvoicingService{},
		connectionSvc:    &stubConnService{},
		numerationSvc:    numeration,
		importSvc:        importSvc,
		testInvoiceSvc:   testSvc,
		sandbox:          sandbox,
		webhookSecret:    "secret",
		logger:           nopHandlerLogger{},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		auth.SetIdentity(c, &auth.Identity{
			UserID: "m", Email: "a@b.co", EmailVerified: true, OrganizationID: "o5",
			Roles: []auth.Role{auth.RoleAdmin},
			Permissions: []auth.Permission{auth.PermOrgManage, auth.PermResourceView},
			ExpiresAt:    time.Now().Add(time.Hour),
		})
		auth.SetRequestContext(c, &auth.RequestContext{OrganizationID: 5, AccountID: 1})
		c.Next()
	})
	r.GET("/api/v1/org/siigo/numeration", h.GetNumeration)
	r.POST("/api/v1/org/siigo/confirm-numeration", h.ConfirmNumeration)
	r.GET("/api/v1/org/siigo/import/preview", h.PreviewImport)
	r.POST("/api/v1/org/siigo/import/confirm", h.ConfirmImport)
	r.POST("/api/v1/org/siigo/sync", h.SyncCustomers)
	r.POST("/api/v1/org/siigo/test-invoice", h.TestInvoice)
	return &onboardingDataTestHandler{h: h, r: r}
}

func (t *onboardingDataTestHandler) do(method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	t.r.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader("")))
	return w
}

func TestGetNumeration_AutoMode(t *testing.T) {
	th := newOnboardingDataTestHandler(
		&stubNumerationSvc{info: &domain.NumerationInfo{Mode: domain.NumerationAuto}},
		nil, nil, true,
	)
	w := th.do(http.MethodGet, "/api/v1/org/siigo/numeration")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"auto"`) {
		t.Fatalf("expected auto mode: %s", w.Body.String())
	}
}

func TestConfirmNumeration_Advances(t *testing.T) {
	th := newOnboardingDataTestHandler(&stubNumerationSvc{}, nil, nil, true)
	w := th.do(http.MethodPost, "/api/v1/org/siigo/confirm-numeration")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "auto") {
		t.Fatalf("expected mode in response: %s", w.Body.String())
	}
}

func TestPreviewImport_ReturnsCounts(t *testing.T) {
	counts := &services.ImportCounts{Total: 5, Nuevos: 2, Existentes: 1, Duplicados: 1, SinNit: 1}
	th := newOnboardingDataTestHandler(nil, &stubImportSvc{counts: counts}, nil, true)
	w := th.do(http.MethodGet, "/api/v1/org/siigo/import/preview")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var payload struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Data["nuevos"] != float64(2) || payload.Data["total"] != float64(5) {
		t.Fatalf("unexpected preview payload: %v", payload.Data)
	}
}

func TestImportConfirmAndSync(t *testing.T) {
	th := newOnboardingDataTestHandler(nil, &stubImportSvc{counts: &services.ImportCounts{Nuevos: 3}}, nil, true)
	if w := th.do(http.MethodPost, "/api/v1/org/siigo/import/confirm"); w.Code != http.StatusOK {
		t.Fatalf("confirm expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if w := th.do(http.MethodPost, "/api/v1/org/siigo/sync"); w.Code != http.StatusOK {
		t.Fatalf("sync expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTestInvoice_SandboxGuard(t *testing.T) {
	// Production mode: rejected with 400 before any call.
	th := newOnboardingDataTestHandler(nil, nil, &stubTestInvoiceSvc{inv: &domain.Invoice{}}, false)
	w := th.do(http.MethodPost, "/api/v1/org/siigo/test-invoice")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 outside sandbox, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTestInvoice_SandboxSuccess(t *testing.T) {
	th := newOnboardingDataTestHandler(nil, nil, &stubTestInvoiceSvc{
		inv: &domain.Invoice{ExternalID: "inv-test-1", Status: domain.InvoiceStatusValid, Cufe: "CUFE123"},
	}, true)
	w := th.do(http.MethodPost, "/api/v1/org/siigo/test-invoice")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 in sandbox, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "inv-test-1") || !strings.Contains(w.Body.String(), "CUFE123") {
		t.Fatalf("unexpected test invoice response: %s", w.Body.String())
	}
}
