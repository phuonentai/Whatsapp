package crm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
)

type paginatedEnvelope struct {
	Success bool            `json:"success"`
	Total   int32           `json:"total"`
	Data    json.RawMessage `json:"data"`
}

func performPaginatedGET(t *testing.T, handler gin.HandlerFunc, path string, orgID, accountID int32) *httptest.ResponseRecorder {
	t.Helper()
	route := path
	if i := strings.Index(path, "?"); i >= 0 {
		route = path[:i]
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET(route, withAuth(orgID, accountID), handler)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	r.ServeHTTP(w, req)
	return w
}

func decodePaginated(t *testing.T, w *httptest.ResponseRecorder) paginatedEnvelope {
	t.Helper()
	var env paginatedEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	return env
}

func TestListContactosReturnsTotalCount(t *testing.T) {
	fc := &fakeContactService{contacts: []*domain.Contact{
		{ID: 1, PhoneNumber: "573001234567", DisplayName: "Ana"},
		{ID: 2, PhoneNumber: "573009998877", DisplayName: "Luis"},
		{ID: 3, PhoneNumber: "573007778899", DisplayName: "Carla"},
	}}
	h := newCSVTestHandler(fc, nil, nil, &fakeActivityService{})

	w := performPaginatedGET(t, h.ListContactos, "/crm/contactos?limit=25&offset=0", 42, 7)

	require.Equal(t, http.StatusOK, w.Code)
	env := decodePaginated(t, w)
	require.True(t, env.Success)
	require.Equal(t, int32(3), env.Total, "total must reflect the full filtered count, ignoring limit")
	require.NotNil(t, env.Data)
}

func TestListContactosTotalZeroWhenEmpty(t *testing.T) {
	h := newCSVTestHandler(&fakeContactService{}, nil, nil, &fakeActivityService{})

	w := performPaginatedGET(t, h.ListContactos, "/crm/contactos", 42, 7)

	require.Equal(t, http.StatusOK, w.Code)
	env := decodePaginated(t, w)
	require.Equal(t, int32(0), env.Total)
	var items []json.RawMessage
	require.NoError(t, json.Unmarshal(env.Data, &items))
	require.Len(t, items, 0)
}

func TestListEmpresasReturnsTotalCount(t *testing.T) {
	fcomp := &fakeCompanyService{companies: []*domain.CompanyWithCounts{
		{Company: domain.Company{ID: 1, Name: "Tienda"}},
		{Company: domain.Company{ID: 2, Name: "Fabrica"}},
	}}
	h := newCSVTestHandler(nil, fcomp, nil, &fakeActivityService{})

	w := performPaginatedGET(t, h.ListEmpresas, "/crm/empresas?limit=25&offset=0", 42, 7)

	require.Equal(t, http.StatusOK, w.Code)
	env := decodePaginated(t, w)
	require.Equal(t, int32(2), env.Total)
	require.NotNil(t, env.Data)
}

func TestListActividadesReturnsTotalCount(t *testing.T) {
	fact := &fakeActivityService{activities: []*domain.ActivityWithActor{
		{Activity: domain.Activity{ID: 1, Tipo: domain.ActivityTypeNota, Asunto: "Llamada"}},
	}}
	h := newCSVTestHandler(&fakeContactService{}, nil, nil, fact)

	w := performPaginatedGET(t, h.ListActividades, "/crm/actividades?limit=25&offset=0", 42, 7)

	require.Equal(t, http.StatusOK, w.Code)
	env := decodePaginated(t, w)
	require.Equal(t, int32(1), env.Total)
	require.NotNil(t, env.Data)
}
