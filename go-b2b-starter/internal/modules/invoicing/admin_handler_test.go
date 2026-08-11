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

	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
)

type stubNumerationRepo struct {
	snapshots map[int32]*domain.NumerationSnapshot
	err       error
}

func (s *stubNumerationRepo) Get(ctx context.Context, orgID int32) (*domain.NumerationSnapshot, error) {
	if s.err != nil {
		return nil, s.err
	}
	if snap, ok := s.snapshots[orgID]; ok {
		return snap, nil
	}
	return nil, domain.ErrConnectionNotFound
}

func (s *stubNumerationRepo) UpsertConfirmed(ctx context.Context, snapshot *domain.NumerationSnapshot) (*domain.NumerationSnapshot, error) {
	return snapshot, nil
}

type stubImportRunRepo struct {
	runs map[int32][]*domain.ImportRun
	err  error
}

func (s *stubImportRunRepo) Record(ctx context.Context, run *domain.ImportRun) (*domain.ImportRun, error) {
	return run, nil
}

func (s *stubImportRunRepo) ListByOrg(ctx context.Context, orgID int32, limit int32) ([]*domain.ImportRun, error) {
	if s.err != nil {
		return nil, s.err
	}
	runs := s.runs[orgID]
	if len(runs) > int(limit) {
		runs = runs[:limit]
	}
	return runs, nil
}

func newAdminListHandler() *Handler {
	now := time.Now()
	connSvc := &stubConnService{}
	h := &Handler{
		invoicingService: &stubInvoicingService{},
		connectionSvc:    connSvc,
		numerationRepo: &stubNumerationRepo{snapshots: map[int32]*domain.NumerationSnapshot{
			5: {OrganizationID: 5, Mode: domain.NumerationAuto, Prefix: "FAC1", NextNumber: "124", ConfirmedAt: &now},
		}},
		importRunRepo: &stubImportRunRepo{runs: map[int32][]*domain.ImportRun{
			5: {{OrganizationID: 5, Kind: domain.ImportRunDelta, Counts: map[string]int32{"nuevos": 3}, PulledAt: now}},
		}},
		logger: nopHandlerLogger{},
	}
	// stubConnService.StatusAll returns org 5 connected (defined in conn_handler_test).
	return h
}

func TestAdminListConnections_AggregatesNumerationsAndRuns(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newAdminListHandler()
	r := gin.New()
	r.GET("/v1/admin/siigo/connections", h.AdminListConnections)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/admin/siigo/connections", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	raw := w.Body.String()
	if !strings.Contains(raw, `"numeration"`) || !strings.Contains(raw, "FAC1") || !strings.Contains(raw, "124") {
		t.Fatalf("expected numeration snapshot in rows: %s", raw)
	}
	if !strings.Contains(raw, `"last_import_run"`) || !strings.Contains(raw, "delta") {
		t.Fatalf("expected last import run in rows: %s", raw)
	}
	if strings.Contains(raw, "client_secret_enc") || strings.Contains(raw, "client_id_enc") {
		t.Fatal("admin list leaked credential columns")
	}
}

func TestAdminListConnections_ToleratesMissingNumerations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// stubConnService.StatusAll returns one org (5); numeration/run repos only
	// know org 5, so the row aggregates; a second org without data is fine.
	h := &Handler{
		invoicingService: &stubInvoicingService{},
		connectionSvc:    &stubConnService{},
		numerationRepo:   &stubNumerationRepo{snapshots: map[int32]*domain.NumerationSnapshot{}},
		importRunRepo:    &stubImportRunRepo{runs: map[int32][]*domain.ImportRun{}},
		logger:           nopHandlerLogger{},
	}
	r := gin.New()
	r.GET("/v1/admin/siigo/connections", h.AdminListConnections)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/admin/siigo/connections", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var payload struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Data) != 1 {
		t.Fatalf("expected 1 row, got %d", len(payload.Data))
	}
	row := payload.Data[0]
	if _, ok := row["numeracion"]; ok {
		t.Fatal("missing numeration must not inject empty objects")
	}
}
