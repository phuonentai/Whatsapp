package invoicing

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// stubConnService is a configurable ConnectionService for handler tests.
type stubConnService struct {
	status  *domain.OrgConnection
	err     error
	lastReq services.ConnectRequest
}

func (s *stubConnService) Status(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	return s.status, s.err
}

func (s *stubConnService) Connect(ctx context.Context, orgID int32, req services.ConnectRequest) (*domain.OrgConnection, error) {
	s.lastReq = req
	if s.err != nil {
		return nil, s.err
	}
	return &domain.OrgConnection{OrganizationID: orgID, Provider: "siigo", Status: domain.ConnStatusConnected, SiigoCompanyName: "Mi Empresa"}, nil
}

func (s *stubConnService) RequestAssisted(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &domain.OrgConnection{OrganizationID: orgID, Status: domain.ConnStatusAwaitingSetup}, nil
}

func (s *stubConnService) Provision(ctx context.Context, orgID int32, req services.ConnectRequest) (*domain.OrgConnection, error) {
	s.lastReq = req
	if s.err != nil {
		return nil, s.err
	}
	return &domain.OrgConnection{OrganizationID: orgID, Provider: "siigo", Status: domain.ConnStatusConnected}, nil
}

func (s *stubConnService) Pause(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &domain.OrgConnection{OrganizationID: orgID, Status: domain.ConnStatusPaused}, nil
}

func (s *stubConnService) Resume(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &domain.OrgConnection{OrganizationID: orgID, Status: domain.ConnStatusLive}, nil
}

func (s *stubConnService) Activate(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &domain.OrgConnection{OrganizationID: orgID, Status: domain.ConnStatusLive}, nil
}

func (s *stubConnService) Disable(ctx context.Context, orgID int32) (*domain.OrgConnection, error) { return nil, nil }

func (s *stubConnService) ConfirmNumeration(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &domain.OrgConnection{OrganizationID: orgID, Status: domain.ConnStatusNumeracionOK}, nil
}

func (s *stubConnService) ConfirmSandboxOK(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &domain.OrgConnection{OrganizationID: orgID, Status: domain.ConnStatusSandboxOK}, nil
}

func (s *stubConnService) IsLive(ctx context.Context, orgID int32) (bool, error) { return true, nil }

func (s *stubConnService) StatusAll(ctx context.Context) ([]*domain.OrgConnection, error) {
	return []*domain.OrgConnection{{OrganizationID: 5, Provider: "siigo", Status: domain.ConnStatusConnected}}, nil
}

func newConnTestHandler(svc services.ConnectionService) *Handler {
	return &Handler{
		invoicingService: &stubInvoicingService{},
		connectionSvc:    svc,
		webhookSecret:    "secret",
		logger:           nopHandlerLogger{},
	}
}

type nopHandlerLogger struct{}

func (nopHandlerLogger) Debug(msg string, fields ...loggerDomain.Fields) {}
func (nopHandlerLogger) Info(msg string, fields ...loggerDomain.Fields)  {}
func (nopHandlerLogger) Warn(msg string, fields ...loggerDomain.Fields)  {}
func (nopHandlerLogger) Error(msg string, fields ...loggerDomain.Fields) {}
func (nopHandlerLogger) Fatal(msg string, fields ...loggerDomain.Fields) {}
func (nopHandlerLogger) WithFields(fields loggerDomain.Fields) loggerDomain.Logger {
	return nopHandlerLogger{}
}

func doRequest(h *Handler, method, path string, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Identity middleware: admin perms, org 5.
	r.Use(func(c *gin.Context) {
		auth.SetIdentity(c, &authcontext.Identity{
			UserID:         "member-1",
			Email:          "admin@test.local",
			EmailVerified:  true,
			OrganizationID: "org-5",
			Roles:          []auth.Role{auth.RoleAdmin},
			Permissions: []auth.Permission{
				auth.PermOrgManage,
				auth.PermResourceView,
			},
			ExpiresAt: time.Now().Add(time.Hour),
		})
		// org_context middleware equivalent: resolves the DB org id.
		authcontext.SetRequestContext(c, &authcontext.RequestContext{
			OrganizationID: 5,
			AccountID:      1,
		})
		c.Next()
	})
	r.POST("/api/v1/org/siigo/connect", h.ConnectSiigo)
	r.POST("/api/v1/org/siigo/request-assisted", h.RequestAssistedSetup)
	r.POST("/api/v1/org/siigo/pause", h.PauseInvoicing)
	r.POST("/api/v1/org/siigo/resume", h.ResumeInvoicing)
	r.POST("/api/v1/org/siigo/activate", h.ActivateInvoicing)
	r.GET("/api/v1/org/siigo/status", h.GetConnectionStatus)
	r.POST("/api/v1/admin/siigo/provision", h.ProvisionSiigo)

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestGetConnectionStatus_NoSecretsInResponse(t *testing.T) {
	conn := &domain.OrgConnection{
		OrganizationID: 5, Provider: "siigo", Status: domain.ConnStatusConnected,
		ClientIDEnc: []byte("should-never-leak"), ClientSecretEnc: []byte("neither-this"),
	}
	h := newConnTestHandler(&stubConnService{status: conn})

	w := doRequest(h, http.MethodGet, "/api/v1/org/siigo/status", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	raw := w.Body.String()
	if strings.Contains(raw, "should-never-leak") || strings.Contains(raw, "client_secret") || strings.Contains(raw, "client_id_enc") {
		t.Fatalf("response leaked credential material: %s", raw)
	}
	var payload struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Data["status"] != "connected" {
		t.Fatalf("expected connected status, got %v", payload.Data["status"])
	}
}

func TestConnectSiigo_RejectsNitMismatch(t *testing.T) {
	h := newConnTestHandler(&stubConnService{err: domain.ErrNitMismatch})
	w := doRequest(h, http.MethodPost, "/api/v1/org/siigo/connect", `{"client_id":"c","client_secret":"s","nit":"900123"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPauseResume_TogglesStatus(t *testing.T) {
	h := newConnTestHandler(&stubConnService{})

	w := doRequest(h, http.MethodPost, "/api/v1/org/siigo/pause", "")
	if w.Code != http.StatusOK {
		t.Fatalf("pause expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "paused") {
		t.Fatalf("pause response missing paused status: %s", w.Body.String())
	}

	w = doRequest(h, http.MethodPost, "/api/v1/org/siigo/resume", "")
	if w.Code != http.StatusOK {
		t.Fatalf("resume expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "live") {
		t.Fatalf("resume response missing live status: %s", w.Body.String())
	}
}

func TestActivate_SandboxOkToLive(t *testing.T) {
	h := newConnTestHandler(&stubConnService{})
	w := doRequest(h, http.MethodPost, "/api/v1/org/siigo/activate", "")
	if w.Code != http.StatusOK {
		t.Fatalf("activate expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "live") {
		t.Fatalf("activate response missing live status: %s", w.Body.String())
	}
}

func TestProvisionSiigo_ForwardsOrgAndCreds(t *testing.T) {
	svc := &stubConnService{}
	h := newConnTestHandler(svc)

	w := doRequest(h, http.MethodPost, "/api/v1/admin/siigo/provision", `{"organization_id":42,"client_id":"cid","client_secret":"csec","nit":"900111"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("provision expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if svc.lastReq.ClientID != "cid" || svc.lastReq.ClientSecret != "csec" || svc.lastReq.Nit != "900111" {
		t.Fatalf("provision did not forward request: %+v", svc.lastReq)
	}
	if strings.Contains(w.Body.String(), "csec") || strings.Contains(w.Body.String(), "cid") {
		t.Fatal("provision response leaked credentials")
	}
}

func TestProvisionSiigo_InvalidTransitionConflict(t *testing.T) {
	h := newConnTestHandler(&stubConnService{err: domain.ErrInvalidTransition})
	w := doRequest(h, http.MethodPost, "/api/v1/admin/siigo/provision", `{"organization_id":42,"client_id":"c","client_secret":"s","nit":"1"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for invalid transition, got %d: %s", w.Code, w.Body.String())
	}
}

func TestProvisionSiigo_GenericError500(t *testing.T) {
	h := newConnTestHandler(&stubConnService{err: errors.New("boom")})
	w := doRequest(h, http.MethodPost, "/api/v1/admin/siigo/provision", `{"organization_id":42,"client_id":"c","client_secret":"s","nit":"1"}`)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}
