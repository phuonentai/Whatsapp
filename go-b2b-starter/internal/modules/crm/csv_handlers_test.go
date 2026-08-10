package crm

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
)

// ---- fakes ----

type fakeLogger struct{}

func (f *fakeLogger) Debug(msg string, fields ...logger.Fields)     {}
func (f *fakeLogger) Info(msg string, fields ...logger.Fields)      {}
func (f *fakeLogger) Warn(msg string, fields ...logger.Fields)      {}
func (f *fakeLogger) Error(msg string, fields ...logger.Fields)     {}
func (f *fakeLogger) Fatal(msg string, fields ...logger.Fields)     {}
func (f *fakeLogger) WithFields(fields logger.Fields) logger.Logger { return f }

type fakeContactService struct {
	contacts []*domain.Contact
	byPhone  map[string]*domain.Contact
	created  []*services.CreateContactRequest
	err      error
}

func (f *fakeContactService) Create(ctx context.Context, orgID int32, req *services.CreateContactRequest) (*domain.Contact, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.created = append(f.created, req)
	return &domain.Contact{ID: int32(len(f.created)), OrganizationID: orgID, PhoneNumber: req.PhoneNumber}, nil
}
func (f *fakeContactService) GetByID(ctx context.Context, orgID, contactID int32) (*domain.Contact, error) {
	return nil, domain.ErrContactNotFound
}
func (f *fakeContactService) GetByPhone(ctx context.Context, orgID int32, phoneNumber string) (*domain.Contact, error) {
	if c, ok := f.byPhone[phoneNumber]; ok {
		return c, nil
	}
	return nil, domain.ErrContactNotFound
}
func (f *fakeContactService) List(ctx context.Context, orgID int32, source, leadStatus string, companyID, assignedTo, limit, offset int32) ([]*domain.Contact, error) {
	return f.contacts, nil
}
func (f *fakeContactService) Search(ctx context.Context, orgID int32, query string, limit, offset int32) ([]*domain.Contact, error) {
	return nil, nil
}
func (f *fakeContactService) Update(ctx context.Context, orgID int32, req *services.UpdateContactRequest) (*domain.Contact, error) {
	return nil, nil
}
func (f *fakeContactService) Delete(ctx context.Context, orgID, contactID int32) error { return nil }

type fakeCompanyService struct {
	companies []*domain.CompanyWithCounts
	byName    map[string]*domain.CompanyWithCounts
}

func (f *fakeCompanyService) Create(ctx context.Context, orgID int32, req *services.CreateCompanyRequest) (*domain.Company, error) {
	return nil, nil
}
func (f *fakeCompanyService) GetByID(ctx context.Context, orgID, companyID int32) (*domain.CompanyWithCounts, error) {
	return nil, domain.ErrCompanyNotFound
}
func (f *fakeCompanyService) List(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.CompanyWithCounts, error) {
	return f.companies, nil
}
func (f *fakeCompanyService) Search(ctx context.Context, orgID int32, query string, limit, offset int32) ([]*domain.CompanyWithCounts, error) {
	if c, ok := f.byName[strings.ToLower(strings.TrimSpace(query))]; ok {
		return []*domain.CompanyWithCounts{c}, nil
	}
	return nil, nil
}
func (f *fakeCompanyService) Update(ctx context.Context, orgID int32, req *services.UpdateCompanyRequest) (*domain.Company, error) {
	return nil, nil
}
func (f *fakeCompanyService) Delete(ctx context.Context, orgID, companyID int32) error { return nil }

type fakeDealService struct {
	deals []*domain.DealWithRefs
}

func (f *fakeDealService) Create(ctx context.Context, orgID int32, req *services.CreateDealRequest) (*domain.Deal, error) {
	return nil, nil
}
func (f *fakeDealService) GetByID(ctx context.Context, orgID, dealID int32) (*domain.DealWithRefs, error) {
	return nil, domain.ErrDealNotFound
}
func (f *fakeDealService) List(ctx context.Context, orgID int32, pipelineID, stageID int32, estado string, contactID, limit, offset int32) ([]*domain.DealWithRefs, error) {
	return f.deals, nil
}
func (f *fakeDealService) Update(ctx context.Context, orgID int32, req *services.UpdateDealRequest) (*domain.Deal, error) {
	return nil, nil
}
func (f *fakeDealService) UpdateStage(ctx context.Context, orgID, dealID, stageID, changedBy int32, oldStageName, newStageName string) (*domain.Deal, error) {
	return nil, nil
}
func (f *fakeDealService) Delete(ctx context.Context, orgID, dealID int32) error { return nil }

type fakeActivityService struct {
	activities []*domain.ActivityWithActor
	created    []*services.CreateActivityRequest
}

func (f *fakeActivityService) Create(ctx context.Context, orgID int32, req *services.CreateActivityRequest) (*domain.Activity, error) {
	f.created = append(f.created, req)
	return &domain.Activity{ID: int32(len(f.created)), OrganizationID: orgID, Tipo: req.Tipo}, nil
}
func (f *fakeActivityService) ListByOrganization(ctx context.Context, orgID int32, tipo, entityType string, entityID, limit, offset int32) ([]*domain.ActivityWithActor, error) {
	return f.activities, nil
}
func (f *fakeActivityService) ListByContact(ctx context.Context, contactID, orgID int32, limit, offset int32) ([]*domain.ActivityWithActor, error) {
	return nil, nil
}
func (f *fakeActivityService) ListByDeal(ctx context.Context, dealID, orgID int32, limit, offset int32) ([]*domain.ActivityWithActor, error) {
	return nil, nil
}
func (f *fakeActivityService) ListByCompany(ctx context.Context, companyID, orgID int32, limit, offset int32) ([]*domain.ActivityWithActor, error) {
	return nil, nil
}

// ---- harness ----

func newCSVTestHandler(contacts *fakeContactService, companies *fakeCompanyService, deals *fakeDealService, activities *fakeActivityService) *CRMHandler {
	return &CRMHandler{
		contactService:  contacts,
		companyService:  companies,
		dealService:     deals,
		activityService: activities,
		logger:          &fakeLogger{},
	}
}

// withAuth is a test middleware injecting the request context the handlers
// read via auth.GetRequestContext(c).
func withAuth(orgID, accountID int32) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth.SetRequestContext(c, &auth.RequestContext{
			Identity:       &auth.Identity{},
			OrganizationID: orgID,
			AccountID:      accountID,
		})
		c.Next()
	}
}

func performGET(t *testing.T, handler gin.HandlerFunc, path string, orgID, accountID int32) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET(path, withAuth(orgID, accountID), handler)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	r.ServeHTTP(w, req)
	return w
}

func parseCSVBody(t *testing.T, body string) [][]string {
	t.Helper()
	body = strings.TrimPrefix(body, "\ufeff")
	records, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	return records
}

func unwrapData(t *testing.T, body []byte) []byte {
	t.Helper()
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &env))
	return env.Data
}

// ---- export tests ----

func TestExportContactosStreamsCSVWithBOMAndHeaders(t *testing.T) {
	fc := &fakeContactService{contacts: []*domain.Contact{
		{ID: 1, PhoneNumber: "573001234567", DisplayName: "Ana", ConsentStatus: domain.ConsentStatusGranted, Source: domain.ContactSourceWhatsApp, LeadStatus: domain.LeadStatusNuevo},
	}}
	h := newCSVTestHandler(fc, nil, nil, &fakeActivityService{})

	w := performGET(t, h.ExportContactos, "/crm/export/contactos.csv", 42, 7)

	require.Equal(t, http.StatusOK, w.Code)
	require.True(t, bytes.HasPrefix(w.Body.Bytes(), []byte{0xEF, 0xBB, 0xBF}), "response must start with UTF-8 BOM")
	require.Equal(t, "text/csv; charset=utf-8", w.Header().Get("Content-Type"))
	require.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
	require.Contains(t, w.Header().Get("Content-Disposition"), "contactos.csv")

	records := parseCSVBody(t, w.Body.String())
	require.Len(t, records, 2)
	require.Equal(t, "Nombre", records[0][1])
	require.Equal(t, "573001234567", records[1][2])
}

func TestExportContactosAuditsSistemaActivity(t *testing.T) {
	fc := &fakeContactService{contacts: []*domain.Contact{
		{ID: 1, PhoneNumber: "573001234567", DisplayName: "Ana"},
		{ID: 2, PhoneNumber: "573009998877", DisplayName: "Luis"},
	}}
	fa := &fakeActivityService{}
	h := newCSVTestHandler(fc, nil, nil, fa)

	w := performGET(t, h.ExportContactos, "/crm/export/contactos.csv", 42, 7)
	require.Equal(t, http.StatusOK, w.Code)

	require.Len(t, fa.created, 1)
	require.Equal(t, domain.ActivityTypeSistema, fa.created[0].Tipo)
	require.Equal(t, "Exportación de contactos", fa.created[0].Asunto)
	require.NotNil(t, fa.created[0].RealizadaPor)
	require.Equal(t, int32(7), *fa.created[0].RealizadaPor)
}

func TestExportContactosMasksWithdrawnConsent(t *testing.T) {
	fc := &fakeContactService{contacts: []*domain.Contact{
		{ID: 1, PhoneNumber: "573001234567", DisplayName: "Ana", Email: "ana@correo.co",
			TipoDocumento: domain.TipoDocCC, NumeroDocumento: "123", ConsentStatus: domain.ConsentStatusWithdrawn},
	}}
	h := newCSVTestHandler(fc, nil, nil, &fakeActivityService{})

	w := performGET(t, h.ExportContactos, "/crm/export/contactos.csv", 1, 1)
	records := parseCSVBody(t, w.Body.String())
	require.Equal(t, "[TELEFONO]", records[1][2])
	require.Equal(t, "[NOMBRE]", records[1][1])
	require.Equal(t, "[EMAIL]", records[1][3])
	require.Equal(t, "[DOCUMENTO]", records[1][4])
}

func TestExportSanitizesFormulaInjection(t *testing.T) {
	fc := &fakeContactService{contacts: []*domain.Contact{
		{ID: 1, DisplayName: "=HYPERLINK(\"http://evil\")", PhoneNumber: "57300"},
	}}
	h := newCSVTestHandler(fc, nil, nil, &fakeActivityService{})

	w := performGET(t, h.ExportContactos, "/crm/export/contactos.csv", 1, 1)
	require.Contains(t, w.Body.String(), "'=HYPERLINK")
}

func TestExportEmpresasNegociosActividades(t *testing.T) {
	fcomp := &fakeCompanyService{companies: []*domain.CompanyWithCounts{{Company: domain.Company{ID: 1, Name: "Tienda"}}}}
	fdeal := &fakeDealService{deals: []*domain.DealWithRefs{{Deal: domain.Deal{ID: 1, Nombre: "Negocio A", Moneda: "COP"}}}}
	fact := &fakeActivityService{activities: []*domain.ActivityWithActor{{Activity: domain.Activity{ID: 1, Tipo: domain.ActivityTypeNota, Asunto: "Llamada"}}}}

	cases := []struct {
		handler gin.HandlerFunc
		path    string
	}{
		{newCSVTestHandler(&fakeContactService{}, fcomp, fdeal, fact).ExportEmpresas, "/crm/export/empresas.csv"},
		{newCSVTestHandler(&fakeContactService{}, fcomp, fdeal, fact).ExportNegocios, "/crm/export/negocios.csv"},
		{newCSVTestHandler(&fakeContactService{}, fcomp, fdeal, fact).ExportActividades, "/crm/export/actividades.csv"},
	}
	for _, tc := range cases {
		w := performGET(t, tc.handler, tc.path, 1, 1)
		require.Equal(t, http.StatusOK, w.Code, tc.path)
		require.True(t, bytes.HasPrefix(w.Body.Bytes(), []byte{0xEF, 0xBB, 0xBF}), tc.path)
		require.Equal(t, "text/csv; charset=utf-8", w.Header().Get("Content-Type"), tc.path)
	}
}

// ---- template test ----

func TestImportContactosTemplate(t *testing.T) {
	h := newCSVTestHandler(&fakeContactService{}, &fakeCompanyService{}, &fakeDealService{}, &fakeActivityService{})
	w := performGET(t, h.ImportContactosTemplate, "/crm/import/contactos/template.csv", 1, 1)

	require.Equal(t, http.StatusOK, w.Code)
	require.True(t, bytes.HasPrefix(w.Body.Bytes(), []byte{0xEF, 0xBB, 0xBF}))
	records := parseCSVBody(t, w.Body.String())
	require.Len(t, records, 3)
	require.Equal(t, "teléfono", records[0][0])
	require.Equal(t, "nombre", records[0][1])
	require.Equal(t, "573001234567", records[1][0])
}

// ---- import tests ----

func buildCSVUpload(t *testing.T, content string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fw, err := writer.CreateFormFile("file", "contactos.csv")
	require.NoError(t, err)
	_, err = fw.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/crm/import/contactos", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func performImport(t *testing.T, h *CRMHandler, content string, orgID, accountID int32) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/crm/import/contactos", withAuth(orgID, accountID), h.ImportContactos)
	req := buildCSVUpload(t, content)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestImportContactosValidDuplicateAndErrors(t *testing.T) {
	fc := &fakeContactService{byPhone: map[string]*domain.Contact{"57300EXISTE": {ID: 99}}}
	h := newCSVTestHandler(fc, &fakeCompanyService{}, &fakeDealService{}, &fakeActivityService{})

	content := strings.Join([]string{
		"teléfono,nombre,email,tipo_documento,numero_documento,empresa,origen,estado",
		"573001234567,Ana Gómez,ana@correo.co,CC,1012345678,Tienda El Sol,whatsapp,nuevo",
		"57300EXISTE,Duplicado,,,,,,",
		"57300INVALIDO,Sin tipo,,XX,,,,",
		"573001234567,Repetido en archivo,,,,,,",
		"573005555555,Luis,,,,,,",
	}, "\n")

	w := performImport(t, h, content, 5, 7)
	require.Equal(t, http.StatusOK, w.Code)

	var summary services.ImportSummary
	require.NoError(t, json.Unmarshal(unwrapData(t, w.Body.Bytes()), &summary))
	require.Equal(t, int32(2), summary.Importados)
	require.Equal(t, int32(2), summary.Omitidos)
	require.Len(t, summary.Errores, 1)
	require.Equal(t, int32(4), summary.Errores[0].Fila)
	require.Contains(t, summary.Errores[0].Razon, "tipo de documento")
}

func TestImportContactosRejectsNonCSV(t *testing.T) {
	h := newCSVTestHandler(&fakeContactService{}, &fakeCompanyService{}, &fakeDealService{}, &fakeActivityService{})
	w := performImport(t, h, "\x89PNG\r\n\x1a\nbinarygarbage", 1, 1)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestImportContactosRejectsOverRowCap(t *testing.T) {
	h := newCSVTestHandler(&fakeContactService{}, &fakeCompanyService{}, &fakeDealService{}, &fakeActivityService{})
	lines := []string{"teléfono,nombre"}
	for i := 0; i < csvMaxImportRows+1; i++ {
		lines = append(lines, "57300,x")
	}
	w := performImport(t, h, strings.Join(lines, "\n"), 1, 1)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "límite")
}

func TestImportContactosLinksExistingCompany(t *testing.T) {
	fc := &fakeContactService{}
	fcomp := &fakeCompanyService{byName: map[string]*domain.CompanyWithCounts{
		"tienda el sol": {Company: domain.Company{ID: 3, Name: "Tienda El Sol"}},
	}}
	h := newCSVTestHandler(fc, fcomp, &fakeDealService{}, &fakeActivityService{})

	content := "teléfono,nombre,email,tipo_documento,numero_documento,empresa,origen,estado\n573001234567,Ana Gómez,,,,Tienda El Sol,,"
	w := performImport(t, h, content, 5, 7)
	require.Equal(t, http.StatusOK, w.Code)

	require.Len(t, fc.created, 1)
	require.NotNil(t, fc.created[0].CompanyID)
	require.Equal(t, int32(3), *fc.created[0].CompanyID)
	require.Equal(t, domain.ContactSourceImport, fc.created[0].Source)
	require.Equal(t, domain.LeadStatusNuevo, fc.created[0].LeadStatus)
}
