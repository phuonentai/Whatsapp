package crm

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	"github.com/moasq/go-b2b-starter/pkg/response"
)

const (
	// csvMaxUploadBytes caps import upload size (5 MB).
	csvMaxUploadBytes int64 = 5 << 20
	// csvMaxImportRows caps the number of data rows in one import (5000).
	csvMaxImportRows = 5000
)

var (
	validTipoDocumentoImport = map[string]bool{
		string(domain.TipoDocCC):  true,
		string(domain.TipoDocNIT): true,
		string(domain.TipoDocCE):  true,
		string(domain.TipoDocTI):  true,
		string(domain.TipoDocPP):  true,
	}
	validContactSource = map[domain.ContactSource]bool{
		domain.ContactSourceWhatsApp: true,
		domain.ContactSourceManual:   true,
		domain.ContactSourceImport:   true,
		domain.ContactSourceAPI:      true,
	}
	validLeadStatus = map[domain.LeadStatus]bool{
		domain.LeadStatusNuevo:         true,
		domain.LeadStatusContactado:    true,
		domain.LeadStatusCalificado:    true,
		domain.LeadStatusDescalificado: true,
		domain.LeadStatusCliente:       true,
	}
)

// ---- Export handlers ----

func (h *CRMHandler) ExportContactos(c *gin.Context) {
	reqCtx := auth.GetRequestContext(c)
	if reqCtx == nil {
		response.Error(c, http.StatusUnauthorized, "autenticación requerida", nil)
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="contactos.csv"`)

	svc := services.NewExportService()
	rows, err := svc.Stream(c.Request.Context(), c.Writer, services.ContactoCSVHeaders,
		func(ctx context.Context, offset int32) ([][]string, error) {
			contacts, err := h.contactService.List(ctx, reqCtx.OrganizationID, "", "", 0, 0, services.ExportPageSize, offset)
			if err != nil {
				return nil, err
			}
			return services.MapContactoCSV(contacts), nil
		})
	if err != nil {
		h.logger.Error("error al exportar contactos", loggerFields("contactos", reqCtx, err))
		return
	}
	h.recordExportAudit(c, reqCtx, "contactos", rows)
}

func (h *CRMHandler) ExportEmpresas(c *gin.Context) {
	reqCtx := auth.GetRequestContext(c)
	if reqCtx == nil {
		response.Error(c, http.StatusUnauthorized, "autenticación requerida", nil)
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="empresas.csv"`)

	svc := services.NewExportService()
	rows, err := svc.Stream(c.Request.Context(), c.Writer, services.EmpresaCSVHeaders,
		func(ctx context.Context, offset int32) ([][]string, error) {
			companies, err := h.companyService.List(ctx, reqCtx.OrganizationID, services.ExportPageSize, offset)
			if err != nil {
				return nil, err
			}
			return services.MapEmpresaCSV(companies), nil
		})
	if err != nil {
		h.logger.Error("error al exportar empresas", loggerFields("empresas", reqCtx, err))
		return
	}
	h.recordExportAudit(c, reqCtx, "empresas", rows)
}

func (h *CRMHandler) ExportNegocios(c *gin.Context) {
	reqCtx := auth.GetRequestContext(c)
	if reqCtx == nil {
		response.Error(c, http.StatusUnauthorized, "autenticación requerida", nil)
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="negocios.csv"`)

	svc := services.NewExportService()
	rows, err := svc.Stream(c.Request.Context(), c.Writer, services.NegocioCSVHeaders,
		func(ctx context.Context, offset int32) ([][]string, error) {
			deals, err := h.dealService.List(ctx, reqCtx.OrganizationID, 0, 0, "", 0, services.ExportPageSize, offset)
			if err != nil {
				return nil, err
			}
			return services.MapNegocioCSV(deals), nil
		})
	if err != nil {
		h.logger.Error("error al exportar negocios", loggerFields("negocios", reqCtx, err))
		return
	}
	h.recordExportAudit(c, reqCtx, "negocios", rows)
}

func (h *CRMHandler) ExportActividades(c *gin.Context) {
	reqCtx := auth.GetRequestContext(c)
	if reqCtx == nil {
		response.Error(c, http.StatusUnauthorized, "autenticación requerida", nil)
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="actividades.csv"`)

	svc := services.NewExportService()
	rows, err := svc.Stream(c.Request.Context(), c.Writer, services.ActividadCSVHeaders,
		func(ctx context.Context, offset int32) ([][]string, error) {
			activities, err := h.activityService.ListByOrganization(ctx, reqCtx.OrganizationID, "", "", 0, services.ExportPageSize, offset)
			if err != nil {
				return nil, err
			}
			return services.MapActividadCSV(activities), nil
		})
	if err != nil {
		h.logger.Error("error al exportar actividades", loggerFields("actividades", reqCtx, err))
		return
	}
	h.recordExportAudit(c, reqCtx, "actividades", rows)
}

// ---- Import handlers ----

// ImportContactosTemplate serves the exact import template with two example
// rows, gated contact:view.
func (h *CRMHandler) ImportContactosTemplate(c *gin.Context) {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="plantilla-contactos.csv"`)

	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(&buf)
	writer.Write([]string{"teléfono", "nombre", "email", "tipo_documento", "numero_documento", "empresa", "origen", "estado"})
	writer.Write([]string{"573001234567", "Ana María Gómez", "ana@correo.co", "CC", "1012345678", "Tienda El Sol", "whatsapp", "nuevo"})
	writer.Write([]string{"573009998877", "Luis Fernando Pérez", "", "NIT", "900123456", "", "manual", "cliente"})
	writer.Flush()
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())
}

// ImportContactos handles a CSV upload, gated contact:manage.
func (h *CRMHandler) ImportContactos(c *gin.Context) {
	reqCtx := auth.GetRequestContext(c)
	if reqCtx == nil {
		response.Error(c, http.StatusUnauthorized, "autenticación requerida", nil)
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, csvMaxUploadBytes)
	if err := c.Request.ParseMultipartForm(csvMaxUploadBytes); err != nil {
		response.Error(c, http.StatusBadRequest, "Archivo demasiado grande (máximo 5 MB)", err)
		return
	}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Se requiere el archivo CSV en el campo 'file'", err)
		return
	}
	defer file.Close()

	if !csvContentSniff(file) {
		response.Error(c, http.StatusBadRequest, "El archivo no es un CSV válido", nil)
		return
	}

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		response.Error(c, http.StatusBadRequest, "El archivo no es un CSV válido", err)
		return
	}
	if len(records) < 2 {
		response.Error(c, http.StatusBadRequest, "El CSV debe incluir cabecera y al menos una fila", nil)
		return
	}
	if len(records)-1 > csvMaxImportRows {
		response.Error(c, http.StatusBadRequest,
			fmt.Sprintf("El archivo supera el límite de %d filas", csvMaxImportRows), nil)
		return
	}

	cols := importColumnIndex(records[0])

	summary := services.ImportSummary{}
	importedPhones := make(map[string]bool)

	for i := 1; i < len(records); i++ {
		fila := int32(i + 1)
		phone := importCell(records[i], cols["teléfono"], cols["telefono"])
		name := importCell(records[i], cols["nombre"])
		phone = strings.TrimSpace(phone)
		name = strings.TrimSpace(name)

		if phone == "" || name == "" {
			summary.Errores = append(summary.Errores, services.ImportError{Fila: fila, Razon: "teléfono y nombre son requeridos"})
			continue
		}

		tipoDoc := importCell(records[i], cols["tipo_documento"])
		if tipoDoc != "" && !validTipoDocumentoImport[strings.ToUpper(tipoDoc)] {
			summary.Errores = append(summary.Errores, services.ImportError{Fila: fila, Razon: "tipo de documento inválido. Valores: CC, NIT, CE, TI, PP"})
			continue
		}

		origen := strings.TrimSpace(importCell(records[i], cols["origen"]))
		source := domain.ContactSource(origen)
		if origen != "" && !validContactSource[source] {
			summary.Errores = append(summary.Errores, services.ImportError{Fila: fila, Razon: "origen inválido. Valores: whatsapp, manual, api, import"})
			continue
		}
		if origen == "" {
			source = domain.ContactSourceImport
		}

		estado := strings.TrimSpace(importCell(records[i], cols["estado"]))
		leadStatus := domain.LeadStatus(estado)
		if estado != "" && !validLeadStatus[leadStatus] {
			summary.Errores = append(summary.Errores, services.ImportError{Fila: fila, Razon: "estado inválido. Valores: nuevo, contactado, calificado, descalificado, cliente"})
			continue
		}
		if estado == "" {
			leadStatus = domain.LeadStatusNuevo
		}

		// Dedupe by phone: skip, never overwrite (SME files are stale, CRM edits are sovereign).
		existing, err := h.contactService.GetByPhone(c.Request.Context(), reqCtx.OrganizationID, phone)
		if err == nil && existing != nil || importedPhones[phone] {
			summary.Omitidos++
			importedPhones[phone] = true
			continue
		}

		companyID, err := h.resolveCompanyID(c, reqCtx.OrganizationID, importCell(records[i], cols["empresa"]))
		if err != nil {
			summary.Errores = append(summary.Errores, services.ImportError{Fila: fila, Razon: "error al procesar empresa"})
			continue
		}

		req := &services.CreateContactRequest{
			OrganizationID:  reqCtx.OrganizationID,
			PhoneNumber:     phone,
			DisplayName:     name,
			Email:           strings.TrimSpace(importCell(records[i], cols["email"])),
			CompanyID:       companyID,
			Source:          source,
			LeadStatus:      leadStatus,
			TipoDocumento:   domain.TipoDocumento(strings.ToUpper(tipoDoc)),
			NumeroDocumento: strings.TrimSpace(importCell(records[i], cols["numero_documento"])),
		}
		if _, err := h.contactService.Create(c.Request.Context(), reqCtx.OrganizationID, req); err != nil {
			summary.Errores = append(summary.Errores, services.ImportError{Fila: fila, Razon: "no se pudo importar el contacto"})
			continue
		}
		summary.Importados++
		importedPhones[phone] = true
	}

	response.Success(c, http.StatusOK, summary)
}

func (h *CRMHandler) resolveCompanyID(c *gin.Context, orgID int32, name string) (*int32, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	companies, err := h.companyService.Search(c.Request.Context(), orgID, name, 1, 0)
	if err != nil {
		return nil, err
	}
	if len(companies) == 0 {
		return nil, nil
	}
	id := companies[0].ID
	return &id, nil
}

// ---- Audit ----

func (h *CRMHandler) recordExportAudit(c *gin.Context, reqCtx *auth.RequestContext, entity string, rows int) {
	realizadaPor := reqCtx.AccountID
	_, err := h.activityService.Create(c.Request.Context(), reqCtx.OrganizationID, &services.CreateActivityRequest{
		Tipo:         domain.ActivityTypeSistema,
		Asunto:       fmt.Sprintf("Exportación de %s", entity),
		Contenido:    fmt.Sprintf("Exportación de %s: %d registros", entity, rows),
		RealizadaPor: &realizadaPor,
		Metadata:     map[string]interface{}{"entity": entity, "rows": rows},
	})
	if err != nil {
		h.logger.Error("error al registrar auditoría de exportación", loggerFields(entity, reqCtx, err))
	}
}

// ---- helpers ----

func csvContentSniff(file multipart.File) bool {
	head := make([]byte, 512)
	n, err := file.Read(head)
	if err != nil && err.Error() != "EOF" {
		return false
	}
	head = head[:n]
	// Reset the file so the CSV parser can re-read from the start.
	if _, err := file.Seek(0, 0); err != nil {
		return false
	}
	if len(bytes.TrimSpace(head)) == 0 {
		return false
	}
	ctype := http.DetectContentType(head)
	switch {
	case strings.HasPrefix(ctype, "text/"):
		return true
	case ctype == "application/csv", ctype == "application/vnd.ms-excel":
		return true
	default:
		return false
	}
}

func importColumnIndex(header []string) map[string]int {
	idx := make(map[string]int)
	for i, col := range header {
		key := normalizeHeader(col)
		if _, exists := idx[key]; !exists {
			idx[key] = i
		}
	}
	return idx
}

func normalizeHeader(col string) string {
	col = strings.ToLower(strings.TrimSpace(col))
	replacer := strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u")
	return replacer.Replace(col)
}

func importCell(row []string, cols ...int) string {
	for _, idx := range cols {
		if idx >= 0 && idx < len(row) {
			return row[idx]
		}
	}
	return ""
}

func loggerFields(entity string, reqCtx *auth.RequestContext, err error) map[string]interface{} {
	return map[string]interface{}{
		"entity":          entity,
		"organization_id": strconv.Itoa(int(reqCtx.OrganizationID)),
		"error":           err.Error(),
	}
}
