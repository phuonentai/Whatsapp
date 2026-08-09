package playbooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playbooksServices "github.com/moasq/go-b2b-starter/internal/modules/playbooks/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/playbooks/domain"
)

type fakeService struct {
	catalog  []*playbooksServices.CatalogEntry
	applyErr error
	resetErr error
	applied  string
}

func (f *fakeService) ListCatalog(ctx context.Context, orgID int32) ([]*playbooksServices.CatalogEntry, error) {
	return f.catalog, nil
}
func (f *fakeService) Apply(ctx context.Context, orgID int32, key string) (*domain.OrganizationPlaybook, error) {
	if f.applyErr != nil {
		return nil, f.applyErr
	}
	f.applied = key
	return &domain.OrganizationPlaybook{OrganizationID: orgID, PlaybookKey: key, AppliedAt: "2026-01-01"}, nil
}
func (f *fakeService) Reset(ctx context.Context, orgID int32, key string) error {
	if f.resetErr != nil {
		return f.resetErr
	}
	f.applied = ""
	return nil
}

func newTestRouter(t *testing.T, svc playbooksServices.PlaybookService) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(svc)
	r.GET("/api/playbooks", h.GetCatalog)
	r.POST("/api/playbooks/:key/apply", h.Apply)
	r.POST("/api/playbooks/:key/reset", h.Reset)
	return r
}

func TestGetCatalogReturnsPlaybooksWithAppliedGuiones(t *testing.T) {
	svc := &fakeService{catalog: []*playbooksServices.CatalogEntry{
		{
			Playbook: &domain.Playbook{Key: "comercio", Name: "Comercio & E-commerce", Vertical: "retail", Description: "desc"},
			Applied:  &domain.OrganizationPlaybook{PlaybookKey: "comercio", AppliedAt: "2026-01-01"},
			Guiones:  []domain.Guion{{ID: "saludo", Titulo: "Saludo", Mensaje: "Hola"}},
		},
		{
			Playbook: &domain.Playbook{Key: "talleres", Name: "Talleres & Reparación", Vertical: "talleres", Description: "desc"},
		},
	}}
	router := newTestRouter(t, svc)

	req := httptest.NewRequest(http.MethodGet, "/api/playbooks", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data, 2)
	assert.Equal(t, "comercio", body.Data[0]["key"])
	assert.Equal(t, true, body.Data[0]["applied"])
	assert.Len(t, body.Data[0]["guiones"], 1)
	assert.Equal(t, "saludo", body.Data[0]["guiones"].([]any)[0].(map[string]any)["id"])
	assert.Equal(t, false, body.Data[1]["applied"])
	_, hasGuiones := body.Data[1]["guiones"]
	assert.False(t, hasGuiones, "non-applied playbook must not expose guiones")
}

func TestApplyReturns404ForUnknownPlaybook(t *testing.T) {
	svc := &fakeService{applyErr: domain.ErrPlaybookNotFound}
	router := newTestRouter(t, svc)

	req := httptest.NewRequest(http.MethodPost, "/api/playbooks/nope/apply", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "Playbook no encontrado")
}

func TestApplyReturns400ForInvalidPayload(t *testing.T) {
	svc := &fakeService{applyErr: domain.ErrInvalidPlaybookPayload}
	router := newTestRouter(t, svc)

	req := httptest.NewRequest(http.MethodPost, "/api/playbooks/comercio/apply", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "No se pudo aplicar el playbook")
}

func TestResetReturns404ForUnknownPlaybook(t *testing.T) {
	svc := &fakeService{resetErr: domain.ErrPlaybookNotFound}
	router := newTestRouter(t, svc)

	req := httptest.NewRequest(http.MethodPost, "/api/playbooks/nope/reset", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestApplySuccessResponse(t *testing.T) {
	svc := &fakeService{}
	router := newTestRouter(t, svc)

	req := httptest.NewRequest(http.MethodPost, "/api/playbooks/comercio/apply", strings.NewReader(""))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "comercio", svc.applied)
}
