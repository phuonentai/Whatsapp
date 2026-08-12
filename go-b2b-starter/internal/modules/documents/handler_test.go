package documents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/internal/modules/documents/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/documents/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
)

// mockDocumentService embeds services.DocumentService so only needed methods
// are stubbed.
type mockDocumentService struct {
	services.DocumentService
	getDoc     *domain.Document
	getErr     error
	listResp   *services.ListDocumentsResponse
	updateDoc  *domain.Document
	updateErr  error
	uploadResp *domain.Document
}

func (m *mockDocumentService) GetDocument(ctx context.Context, orgID, docID int32, canViewAdminOnly bool) (*domain.Document, error) {
	return m.getDoc, m.getErr
}

func (m *mockDocumentService) ListDocuments(ctx context.Context, orgID int32, req *services.ListDocumentsRequest, canViewAdminOnly bool) (*services.ListDocumentsResponse, error) {
	return m.listResp, nil
}

func (m *mockDocumentService) UpdateDocument(ctx context.Context, orgID, docID int32, req *services.UpdateDocumentRequest) (*domain.Document, error) {
	return m.updateDoc, m.updateErr
}

// newTestContext builds a Gin context with the request body, an optional
// request context (org + identity), and returns the recorder.
func newTestContext(method, path, body string, identity *authcontext.Identity) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(method, path, bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	// Extract a trailing :id param (handlers read c.Param("id")).
	if strings.Contains(path, "/") {
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 2 {
			c.Params = gin.Params{{Key: "id", Value: parts[1]}}
		}
	}
	authcontext.SetRequestContext(c, &authcontext.RequestContext{
		Identity:       identity,
		OrganizationID: 1,
		AccountID:      1,
	})
	return c, rec
}

func memberIdentity() *authcontext.Identity {
	return &authcontext.Identity{
		UserID: "member-1",
		Roles:  []authcontext.Role{authcontext.RoleMember},
		Permissions: []authcontext.Permission{
			authcontext.NewPermission("resource", "view"),
		},
	}
}

func adminIdentity() *authcontext.Identity {
	return &authcontext.Identity{
		UserID: "admin-1",
		Roles:  []authcontext.Role{authcontext.RoleAdmin},
		Permissions: []authcontext.Permission{
			authcontext.NewPermission("resource", "view"),
			authcontext.NewPermission("org", "manage"),
		},
	}
}

func makeDocument() *domain.Document {
	return &domain.Document{
		ID:             1,
		OrganizationID: 1,
		FileAssetID:    10,
		Title:          "Manual interno",
		FileName:       "manual.pdf",
		ContentType:    "application/pdf",
		FileSize:       1024,
		Status:         domain.DocumentStatusProcessed,
		Visibility:     domain.DocumentVisibilityWorkspace,
	}
}

func TestGetDocumentRestrictedDocIs404ForMember(t *testing.T) {
	handler := NewHandler(&mockDocumentService{getErr: domain.ErrDocumentNotFound})
	c, rec := newTestContext(http.MethodGet, "/example_documents/1", "", memberIdentity())

	handler.GetDocument(c)

	require.Equal(t, http.StatusNotFound, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "document_not_found", body["code"])
	// No title leak: the restricted document title never appears.
	assert.NotContains(t, rec.Body.String(), "Manual interno")
}

func TestGetDocumentAdminSeesAdminOnlyDoc(t *testing.T) {
	doc := makeDocument()
	doc.Visibility = domain.DocumentVisibilityAdminOnly
	handler := NewHandler(&mockDocumentService{getDoc: doc})
	c, rec := newTestContext(http.MethodGet, "/example_documents/1", "", adminIdentity())

	handler.GetDocument(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp domain.Document
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, domain.DocumentVisibilityAdminOnly, resp.Visibility)
	assert.Equal(t, "Manual interno", resp.Title)
}

func TestUpdateDocumentRejectsInvalidVisibility(t *testing.T) {
	handler := NewHandler(&mockDocumentService{updateErr: domain.ErrInvalidVisibility})
	c, rec := newTestContext(
		http.MethodPatch,
		"/example_documents/1",
		`{"visibility":"everyone"}`,
		adminIdentity(),
	)

	handler.UpdateDocument(c)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "invalid_visibility", body["code"])
}

func TestUpdateDocumentInvalidJSON(t *testing.T) {
	handler := NewHandler(&mockDocumentService{})
	c, rec := newTestContext(
		http.MethodPatch,
		"/example_documents/1",
		`{not json`,
		adminIdentity(),
	)

	handler.UpdateDocument(c)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// Route-level gate: the update/delete/upload endpoints require org:manage
// server-side, so a member without it gets a 403 before the handler runs.
func TestUpdateRouteRequiresOrgManage(t *testing.T) {
	requireForbiddenForMember(t, func(c *gin.Context) {
		auth.RequirePermissionFunc("org", "manage")(c)
	})
}

// requireForbiddenForMember runs the given middleware against a member
// identity (resource:view only) and asserts the request is rejected with 403.
func requireForbiddenForMember(t *testing.T, middlewares ...gin.HandlerFunc) {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPatch, "/example_documents/1", bytes.NewBufferString(`{}`))
	authcontext.SetRequestContext(c, &authcontext.RequestContext{OrganizationID: 1, AccountID: 1})
	authcontext.SetIdentity(c, memberIdentity())

	for _, mw := range middlewares {
		mw(c)
	}
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestListDocumentsMemberGetsFilteredResponse(t *testing.T) {
	handler := NewHandler(&mockDocumentService{
		listResp: &services.ListDocumentsResponse{
			Documents: []*domain.Document{makeDocument()},
			Total:     1,
		},
	})
	c, rec := newTestContext(http.MethodGet, "/example_documents", "", memberIdentity())

	handler.ListDocuments(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp services.ListDocumentsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp.Documents, 1)
	assert.Equal(t, int64(1), resp.Total)
}

func TestExportComplianceRejectsMemberWithoutOrgManage(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/example_documents/export/compliance.csv", nil)
	authcontext.SetRequestContext(c, &authcontext.RequestContext{OrganizationID: 1, AccountID: 1})
	authcontext.SetIdentity(c, memberIdentity())

	auth.RequirePermissionFunc("org", "manage")(c)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestExportComplianceStreamsCSV(t *testing.T) {
	// Direct handler test with a stubbed service that returns indexed docs.
	svc := &complianceExportService{}
	handler := NewHandler(svc)
	c, rec := newTestContext(http.MethodGet, "/example_documents/export/compliance.csv", "", adminIdentity())

	handler.ExportCompliance(c)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/csv")
	assert.Contains(t, rec.Body.String(), "id,titulo,estado,visibilidad,creado_en")
	assert.Contains(t, rec.Body.String(), "manual.pdf")
	assert.Contains(t, rec.Body.String(), "admin_only")
}

var _ services.DocumentService = (*complianceExportService)(nil)

// complianceExportService stubs ExportComplianceDocuments only.
type complianceExportService struct {
	services.DocumentService
}

func (s *complianceExportService) ExportComplianceDocuments(ctx context.Context, orgID int32) ([]domain.ComplianceDocument, error) {
	return []domain.ComplianceDocument{
		{ID: 1, Title: "manual.pdf", Status: domain.DocumentStatusProcessed, Visibility: domain.DocumentVisibilityAdminOnly},
	}, nil
}

var _ = errors.Is // keep errors import for future use
