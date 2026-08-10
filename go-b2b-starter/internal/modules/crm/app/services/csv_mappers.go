package services

import (
	"strconv"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
)

// Masked PII placeholders, consistent with the Habeas Data export invariant.
const (
	maskTelefono  = "[TELEFONO]"
	maskNombre    = "[NOMBRE]"
	maskEmail     = "[EMAIL]"
	maskDocumento = "[DOCUMENTO]"
)

// ContactoCSVHeaders are the Spanish column headers for the contact export,
// matching the CRM list view.
var ContactoCSVHeaders = []string{
	"ID", "Nombre", "Telefono", "Email", "Tipo Documento", "Numero Documento",
	"Empresa ID", "Origen", "Estado", "Cargo", "Creado",
}

// EmpresaCSVHeaders are the Spanish column headers for the company export.
var EmpresaCSVHeaders = []string{
	"ID", "Nombre", "NIT", "Tipo Empresa", "Sector", "Ciudad", "Departamento",
	"Telefono", "Direccion", "Sitio Web", "Notas",
}

// NegocioCSVHeaders are the Spanish column headers for the deal export.
var NegocioCSVHeaders = []string{
	"ID", "Nombre", "Contacto", "Telefono Contacto", "Empresa", "Valor", "Moneda",
	"Estado", "Probabilidad", "Notas", "Cierre Esperado",
}

// ActividadCSVHeaders are the Spanish column headers for the activity export.
var ActividadCSVHeaders = []string{
	"ID", "Tipo", "Asunto", "Contenido", "Estado", "Contacto ID", "Negocio ID",
	"Empresa ID", "Realizada Por", "Realizada En",
}

// MapContactoCSV maps contacts to CSV rows, masking PII for contacts whose
// consent status is withdrawn.
func MapContactoCSV(contacts []*domain.Contact) [][]string {
	rows := make([][]string, 0, len(contacts))
	for _, c := range contacts {
		masked := c.ConsentStatus == domain.ConsentStatusWithdrawn

		phone := c.PhoneNumber
		name := c.DisplayName
		email := c.Email
		tipoDoc := string(c.TipoDocumento)
		numDoc := c.NumeroDocumento

		if masked {
			if phone != "" {
				phone = maskTelefono
			}
			if name != "" {
				name = maskNombre
			}
			if email != "" {
				email = maskEmail
			}
			if tipoDoc != "" || numDoc != "" {
				tipoDoc = maskDocumento
				numDoc = maskDocumento
			}
		}

		rows = append(rows, []string{
			formatID(c.ID), name, phone, email, tipoDoc, numDoc,
			formatIDPtr(c.CompanyID), string(c.Source), string(c.LeadStatus),
			c.JobTitle, formatTime(c.CreatedAt),
		})
	}
	return rows
}

// MapEmpresaCSV maps companies to CSV rows.
func MapEmpresaCSV(companies []*domain.CompanyWithCounts) [][]string {
	rows := make([][]string, 0, len(companies))
	for _, e := range companies {
		rows = append(rows, []string{
			formatID(e.ID), e.Name, e.Nit, e.TipoEmpresa, e.Sector, e.Ciudad,
			e.Departamento, e.Phone, e.Address, e.Website, e.Notes,
		})
	}
	return rows
}

// MapNegocioCSV maps deals to CSV rows.
func MapNegocioCSV(deals []*domain.DealWithRefs) [][]string {
	rows := make([][]string, 0, len(deals))
	for _, d := range deals {
		cierre := ""
		if d.FechaCierreEsperada != nil {
			cierre = formatTime(*d.FechaCierreEsperada)
		}
		rows = append(rows, []string{
			formatID(d.ID), d.Nombre, d.ContactName, d.ContactPhone, d.CompanyName,
			formatFloatPtr(d.Monto), d.Moneda, string(d.Estado), formatIDPtr(d.Probabilidad),
			d.Notas, cierre,
		})
	}
	return rows
}

// MapActividadCSV maps activities to CSV rows.
func MapActividadCSV(activities []*domain.ActivityWithActor) [][]string {
	rows := make([][]string, 0, len(activities))
	for _, a := range activities {
		rows = append(rows, []string{
			formatID(a.ID), string(a.Tipo), a.Asunto, a.Contenido, a.Estado,
			formatIDPtr(a.ContactID), formatIDPtr(a.DealID), formatIDPtr(a.CompanyID),
			a.RealizadaPorNombre, formatTime(a.RealizadaEn),
		})
	}
	return rows
}

func formatID(v int32) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatInt(int64(v), 10)
}

func formatIDPtr(v *int32) string {
	if v == nil {
		return ""
	}
	return strconv.FormatInt(int64(*v), 10)
}

func formatFloatPtr(v *float64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatFloat(*v, 'f', -1, 64)
}

func formatTime(v time.Time) string {
	if v.IsZero() {
		return ""
	}
	return v.UTC().Format(time.RFC3339)
}
