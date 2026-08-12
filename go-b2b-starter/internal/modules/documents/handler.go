package documents

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
	"github.com/moasq/go-b2b-starter/internal/modules/documents/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/documents/domain"
	_ "github.com/moasq/go-b2b-starter/internal/modules/documents/domain" // for swagger
	"github.com/moasq/go-b2b-starter/pkg/httperr"
)

type Handler struct {
	service services.DocumentService
}

func NewHandler(service services.DocumentService) *Handler {
	return &Handler{service: service}
}

// canManage reports whether the request member holds org:manage (the effective
// role available in the request context, populated by the auth middleware).
// Wildcard grants ("*:*" e.g. mock admin) count as matching, mirroring the
// route-level auth.RequirePermissionFunc semantics.
func canManage(c *gin.Context) bool {
	reqCtx := authcontext.GetRequestContext(c)
	if reqCtx == nil || reqCtx.Identity == nil {
		return false
	}
	target := authcontext.NewPermission("org", "manage")
	for _, p := range reqCtx.Identity.Permissions {
		if p == target || p.MatchesWithWildcard(target) {
			return true
		}
	}
	return false
}

// UploadDocument uploads a new PDF document
// @Summary Upload PDF document
// @Description Uploads a PDF document, extracts text, and creates embeddings
// @Tags Documents
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "PDF file to upload"
// @Param title formData string true "Document title"
// @Success 201 {object} domain.Document
// @Failure 400 {object} httperr.HTTPError
// @Failure 500 {object} httperr.HTTPError
// @Router /example_documents/upload [post]
func (h *Handler) UploadDocument(c *gin.Context) {
	reqCtx := authcontext.GetRequestContext(c)
	if reqCtx == nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest,
			"missing_context",
			"Organization context is required",
		))
		return
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest,
			"invalid_file",
			"Failed to read file: "+err.Error(),
		))
		return
	}
	defer file.Close()

	// Get title from form
	title := c.PostForm("title")
	if title == "" {
		title = header.Filename
	}

	// Create upload request
	req := &services.UploadDocumentRequest{
		Title:       title,
		FileName:    header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		FileSize:    header.Size,
	}

	// Upload document
	document, err := h.service.UploadDocument(c.Request.Context(), reqCtx.OrganizationID, req, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError,
			"upload_failed",
			"Failed to upload document: "+err.Error(),
		))
		return
	}

	c.JSON(http.StatusCreated, document)
}

// GetDocument retrieves a single document by ID.
// A restricted document (visibility = admin_only) behaves as nonexistent for
// members without org:manage: 404 with no title leak.
// @Summary Get document
// @Description Gets a document by ID. Restricted documents return 404 for members without org:manage.
// @Tags Documents
// @Produce json
// @Param id path int true "Document ID"
// @Success 200 {object} domain.Document
// @Failure 404 {object} httperr.HTTPError
// @Failure 500 {object} httperr.HTTPError
// @Router /example_documents/{id} [get]
func (h *Handler) GetDocument(c *gin.Context) {
	docID, ok := parseDocumentID(c)
	if !ok {
		return
	}

	reqCtx := authcontext.GetRequestContext(c)
	if reqCtx == nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest,
			"missing_context",
			"Organization context is required",
		))
		return
	}

	document, err := h.service.GetDocument(c.Request.Context(), reqCtx.OrganizationID, docID, canManage(c))
	if err != nil {
		if errors.Is(err, domain.ErrDocumentNotFound) {
			c.JSON(http.StatusNotFound, httperr.NewHTTPError(
				http.StatusNotFound,
				"document_not_found",
				"Document not found",
			))
			return
		}
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError,
			"get_failed",
			"Failed to get document: "+err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, document)
}

// UpdateDocument updates document metadata (title/visibility) — admin-only in v1.
// @Summary Update document
// @Description Updates document title and/or visibility (workspace | admin_only). Requires org:manage.
// @Tags Documents
// @Accept json
// @Produce json
// @Param id path int true "Document ID"
// @Param body body services.UpdateDocumentRequest true "Fields to update"
// @Success 200 {object} domain.Document
// @Failure 400 {object} httperr.HTTPError
// @Failure 404 {object} httperr.HTTPError
// @Failure 500 {object} httperr.HTTPError
// @Router /example_documents/{id} [patch]
func (h *Handler) UpdateDocument(c *gin.Context) {
	docID, ok := parseDocumentID(c)
	if !ok {
		return
	}

	reqCtx := authcontext.GetRequestContext(c)
	if reqCtx == nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest,
			"missing_context",
			"Organization context is required",
		))
		return
	}

	var req services.UpdateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest,
			"invalid_request",
			"Invalid JSON format: "+err.Error(),
		))
		return
	}

	document, err := h.service.UpdateDocument(c.Request.Context(), reqCtx.OrganizationID, docID, &req)
	if err != nil {
		if errors.Is(err, domain.ErrDocumentNotFound) {
			c.JSON(http.StatusNotFound, httperr.NewHTTPError(
				http.StatusNotFound,
				"document_not_found",
				"Document not found",
			))
			return
		}
		if errors.Is(err, domain.ErrInvalidVisibility) {
			c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
				http.StatusBadRequest,
				"invalid_visibility",
				err.Error(),
			))
			return
		}
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError,
			"update_failed",
			"Failed to update document: "+err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, document)
}

// ListDocuments lists documents with pagination
// @Summary List documents
// @Description Lists documents with optional filtering and pagination
// @Tags Documents
// @Produce json
// @Param limit query int false "Limit" default(10)
// @Param offset query int false "Offset" default(0)
// @Param status query string false "Filter by status (pending, processing, processed, failed)"
// @Success 200 {object} services.ListDocumentsResponse
// @Failure 500 {object} httperr.HTTPError
// @Router /example_documents [get]
func (h *Handler) ListDocuments(c *gin.Context) {
	reqCtx := authcontext.GetRequestContext(c)
	if reqCtx == nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest,
			"missing_context",
			"Organization context is required",
		))
		return
	}

	// Parse query parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	req := &services.ListDocumentsRequest{
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	// Optional status filter
	// Note: Status filtering would need to be added if needed

	response, err := h.service.ListDocuments(c.Request.Context(), reqCtx.OrganizationID, req, canManage(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError,
			"list_failed",
			"Failed to list documents: "+err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, response)
}

// parseDocumentID parses the :id path param into an int32, writing a 400
// response when the value is not numeric.
func parseDocumentID(c *gin.Context) (int32, bool) {
	idParam := c.Param("id")
	var docID int32
	if _, err := fmt.Sscanf(idParam, "%d", &docID); err != nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest,
			"invalid_id",
			"Document ID must be a valid number",
		))
		return 0, false
	}
	return docID, true
}

// @Summary Delete document
// @Description Deletes a document and its associated file
// @Tags Documents
// @Param id path int true "Document ID"
// @Success 204
// @Failure 400 {object} httperr.HTTPError
// @Failure 500 {object} httperr.HTTPError
// @Router /example_documents/{id} [delete]
func (h *Handler) DeleteDocument(c *gin.Context) {
	docID, ok := parseDocumentID(c)
	if !ok {
		return
	}

	reqCtx := authcontext.GetRequestContext(c)
	if reqCtx == nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest,
			"missing_context",
			"Organization context is required",
		))
		return
	}

	if err := h.service.DeleteDocument(c.Request.Context(), reqCtx.OrganizationID, docID); err != nil {
		if errors.Is(err, domain.ErrDocumentNotFound) {
			c.JSON(http.StatusNotFound, httperr.NewHTTPError(
				http.StatusNotFound,
				"document_not_found",
				"Document not found",
			))
			return
		}
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError,
			"delete_failed",
			"Failed to delete document: "+err.Error(),
		))
		return
	}

	c.Status(http.StatusNoContent)
}

// ExportCompliance streams the org's indexed documents (Ley 1581 traceability)
// as CSV: title, status, visibility, created_at. Admin-only (org:manage).
// @Summary Export indexed documents (compliance)
// @Description Lists documents that contributed chunks to the RAG index with their visibility, for Ley 1581 traceability.
// @Tags Documents
// @Produce text/csv
// @Success 200 {string} string
// @Failure 500 {object} httperr.HTTPError
// @Router /example_documents/export/compliance.csv [get]
func (h *Handler) ExportCompliance(c *gin.Context) {
	reqCtx := authcontext.GetRequestContext(c)
	if reqCtx == nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest,
			"missing_context",
			"Organization context is required",
		))
		return
	}

	docs, err := h.service.ExportComplianceDocuments(c.Request.Context(), reqCtx.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError,
			"export_failed",
			"Failed to export compliance documents: "+err.Error(),
		))
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="documentos-indexados.csv"`)

	// UTF-8 BOM so Spanish accents render correctly in Excel.
	c.Writer.WriteString("\xEF\xBB\xBF")
	c.Writer.WriteString("id,titulo,estado,visibilidad,creado_en\n")
	for _, doc := range docs {
		fmt.Fprintf(c.Writer, "%d,%s,%s,%s,%s\n",
			doc.ID,
			csvEscape(doc.Title),
			doc.Status,
			doc.Visibility,
			doc.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	}
}

// ReprocessDocument re-runs the document processing pipeline (re-extract text
// and re-embed chunks) — used by the Retry action. Never re-uploads the file.
// @Summary Reprocess document
// @Description Re-runs extraction and re-embeds chunks for a document. Requires org:manage.
// @Tags Documents
// @Param id path int true "Document ID"
// @Success 200 {object} domain.Document
// @Failure 404 {object} httperr.HTTPError
// @Failure 500 {object} httperr.HTTPError
// @Router /example_documents/{id}/reprocess [post]
func (h *Handler) ReprocessDocument(c *gin.Context) {
	docID, ok := parseDocumentID(c)
	if !ok {
		return
	}

	reqCtx := authcontext.GetRequestContext(c)
	if reqCtx == nil {
		c.JSON(http.StatusBadRequest, httperr.NewHTTPError(
			http.StatusBadRequest,
			"missing_context",
			"Organization context is required",
		))
		return
	}

	document, err := h.service.ProcessDocument(c.Request.Context(), reqCtx.OrganizationID, docID)
	if err != nil {
		if errors.Is(err, domain.ErrDocumentNotFound) {
			c.JSON(http.StatusNotFound, httperr.NewHTTPError(
				http.StatusNotFound,
				"document_not_found",
				"Document not found",
			))
			return
		}
		c.JSON(http.StatusInternalServerError, httperr.NewHTTPError(
			http.StatusInternalServerError,
			"reprocess_failed",
			"Failed to reprocess document: "+err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, document)
}

// csvEscape quotes and doubles a CSV cell value when it contains separators,
// quotes, or newlines (formula-injection safe: a leading '=' is quoted too).
func csvEscape(value string) string {
	if strings.ContainsAny(value, ",\"\n") || strings.HasPrefix(value, "=") ||
		strings.HasPrefix(value, "+") || strings.HasPrefix(value, "-") || strings.HasPrefix(value, "@") {
		return "\"" + strings.ReplaceAll(value, "\"", "\"\"") + "\""
	}
	return value
}
