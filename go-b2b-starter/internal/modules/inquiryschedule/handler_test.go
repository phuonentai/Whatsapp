package inquiryschedule

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
)

// stubService implements InquiryscheduleService for route tests by embedding
// the interface and overriding the endpoints under test.
type stubService struct {
	services.InquiryscheduleService
	schedule      *domain.Schedule
	statuses      []*domain.ScheduleStatus
	detail        *services.ScheduleDetail
	settings      *domain.FollowUpSettings
	createErr     error
	updateErr     error
	getErr        error
	settingsErr   error
	lastCreateIn  services.CreateScheduleInput
}

func (s *stubService) CreateSchedule(ctx context.Context, orgID int32, memberID string, in services.CreateScheduleInput) (*domain.Schedule, error) {
	s.lastCreateIn = in
	if s.createErr != nil {
		return nil, s.createErr
	}
	if s.schedule != nil {
		return s.schedule, nil
	}
	return &domain.Schedule{
		ID: 1, OrganizationID: orgID, Name: in.Name, RunTime: in.RunTime,
		DaysOfWeek: in.DaysOfWeek, ProductIDs: in.ProductIDs, SupplierIDs: in.SupplierIDs,
		IsActive: true, NextRunAt: time.Now().Add(24 * time.Hour),
	}, nil
}

func (s *stubService) UpdateSchedule(ctx context.Context, orgID int32, memberID string, in services.UpdateScheduleInput) (*domain.Schedule, error) {
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	return &domain.Schedule{ID: in.ID, OrganizationID: orgID, Name: in.Name, NextRunAt: time.Now()}, nil
}

func (s *stubService) PauseSchedule(ctx context.Context, orgID, id int32, memberID string) (*domain.Schedule, error) {
	return &domain.Schedule{ID: id, OrganizationID: orgID, IsActive: false}, nil
}

func (s *stubService) ResumeSchedule(ctx context.Context, orgID, id int32, memberID string) (*domain.Schedule, error) {
	return &domain.Schedule{ID: id, OrganizationID: orgID, IsActive: true, NextRunAt: time.Now()}, nil
}

func (s *stubService) DeleteSchedule(ctx context.Context, orgID, id int32, memberID string) error {
	return nil
}

func (s *stubService) ListSchedules(ctx context.Context, orgID int32) ([]*domain.ScheduleStatus, error) {
	return s.statuses, nil
}

func (s *stubService) GetScheduleDetail(ctx context.Context, orgID, id int32) (*services.ScheduleDetail, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.detail, nil
}

func (s *stubService) GetFollowUpSettings(ctx context.Context, orgID int32) (*domain.FollowUpSettings, error) {
	if s.settingsErr != nil {
		return nil, s.settingsErr
	}
	if s.settings != nil {
		return s.settings, nil
	}
	def := domain.DefaultFollowUpSettings(orgID)
	return &def, nil
}

func (s *stubService) UpdateFollowUpSettings(ctx context.Context, orgID int32, in services.UpdateFollowUpSettingsInput) (*domain.FollowUpSettings, error) {
	if s.settingsErr != nil {
		return nil, s.settingsErr
	}
	return &domain.FollowUpSettings{
		OrganizationID: orgID, Enabled: in.Enabled, DeadlineHours: in.DeadlineHours,
		MaxNudges: in.MaxNudges, MessageTemplate: in.MessageTemplate,
	}, nil
}

// stubResolver provides the named middleware the routes require.
type stubResolver struct {
	perms []string
}

func (r *stubResolver) Get(name string) gin.HandlerFunc {
	switch name {
	case "auth":
		return func(c *gin.Context) {
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

func newRouter(svc services.InquiryscheduleService, resolver *stubResolver) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	routes := NewRoutes(NewHandler(svc))
	api := router.Group("/api")
	routes.RegisterRoutes(api, resolver)
	return router
}

func doRequest(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// ---------- tests ----------

func TestHandler_CreateSchedule_201WithComputedNextRunAt(t *testing.T) {
	svc := &stubService{}
	router := newRouter(svc, &stubResolver{perms: []string{"org:view", "org:manage"}})

	w := doRequest(t, router, http.MethodPost, "/api/procurement/schedules",
		`{"name":"Matinal","run_time":"08:00","days_of_week":[1,2,3,4,5],"product_ids":[1,2],"supplier_ids":[3,4],"note":"cafe"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			ID        int32  `json:"ID"`
			NextRunAt string `json:"NextRunAt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data.ID != 1 {
		t.Fatalf("id = %d, want 1", resp.Data.ID)
	}
	if resp.Data.NextRunAt == "" {
		t.Fatal("next_run_at missing from 201 response")
	}
}

func TestHandler_CreateSchedule_400SpanishValidation(t *testing.T) {
	svc := &stubService{createErr: &domain.ValidationError{Field: "run_time", Message: "La hora de ejecución es requerida."}}
	router := newRouter(svc, &stubResolver{perms: []string{"org:view", "org:manage"}})

	w := doRequest(t, router, http.MethodPost, "/api/procurement/schedules",
		`{"name":"Sin hora","days_of_week":[1]}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("La hora de ejecución es requerida")) {
		t.Fatalf("missing Spanish validation message: %s", w.Body.String())
	}
}

func TestHandler_CreateSchedule_403WithoutManage(t *testing.T) {
	svc := &stubService{}
	router := newRouter(svc, &stubResolver{perms: []string{"org:view"}}) // no org:manage

	w := doRequest(t, router, http.MethodPost, "/api/procurement/schedules",
		`{"name":"Matinal","run_time":"08:00","days_of_week":[1]}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

func TestHandler_GetSchedule_404ForAnotherOrgsSchedule(t *testing.T) {
	svc := &stubService{getErr: domain.ErrScheduleNotFound}
	router := newRouter(svc, &stubResolver{perms: []string{"org:view"}})

	w := doRequest(t, router, http.MethodGet, "/api/procurement/schedules/999", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("no encontrada")) {
		t.Fatalf("missing Spanish 404 message: %s", w.Body.String())
	}
}

func TestHandler_ListSchedules_IncludesNextRunAndLastStatus(t *testing.T) {
	next := time.Now().Add(20 * time.Hour)
	svc := &stubService{statuses: []*domain.ScheduleStatus{{
		Schedule:      &domain.Schedule{ID: 1, OrganizationID: 42, Name: "Matinal", NextRunAt: next, IsActive: true},
		HasLastRun:    true,
		LastRunID:     77,
		LastRunStatus: "awaiting_responses",
	}}}
	router := newRouter(svc, &stubResolver{perms: []string{"org:view"}})

	w := doRequest(t, router, http.MethodGet, "/api/procurement/schedules", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("LastRunStatus")) || !bytes.Contains(w.Body.Bytes(), []byte("awaiting_responses")) {
		t.Fatalf("last run status missing: %s", w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("NextRunAt")) {
		t.Fatalf("next_run_at missing: %s", w.Body.String())
	}
}

func TestHandler_GetSchedule_DetailIncludesRecentRuns(t *testing.T) {
	svc := &stubService{detail: &services.ScheduleDetail{
		Schedule: &domain.Schedule{ID: 1, OrganizationID: 42, Name: "Matinal"},
		FollowUp: func() *domain.FollowUpSettings {
			def := domain.DefaultFollowUpSettings(42)
			return &def
		}(),
		RecentRuns: []*domain.ScheduledRun{{ID: 9, Status: "completed", Source: "scheduled"}},
	}}
	router := newRouter(svc, &stubResolver{perms: []string{"org:view"}})

	w := doRequest(t, router, http.MethodGet, "/api/procurement/schedules/1", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("completed")) {
		t.Fatalf("recent runs missing: %s", w.Body.String())
	}
}

func TestHandler_FollowUpSettings_Update_403WithoutManage(t *testing.T) {
	svc := &stubService{}
	router := newRouter(svc, &stubResolver{perms: []string{"org:view"}})

	w := doRequest(t, router, http.MethodPut, "/api/procurement/followup-settings",
		`{"enabled":true,"deadline_hours":4,"max_nudges":1}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

func TestHandler_FollowUpSettings_Update_400SpanishRange(t *testing.T) {
	svc := &stubService{settingsErr: &domain.ValidationError{Field: "deadline_hours", Message: "Las horas de plazo deben estar entre 1 y 168."}}
	router := newRouter(svc, &stubResolver{perms: []string{"org:view", "org:manage"}})

	w := doRequest(t, router, http.MethodPut, "/api/procurement/followup-settings",
		`{"enabled":true,"deadline_hours":0,"max_nudges":1}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("Las horas de plazo deben estar entre 1 y 168")) {
		t.Fatalf("missing Spanish range message: %s", w.Body.String())
	}
}
