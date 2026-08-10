package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
)

func TestCSVSanitizeCell(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		expected string
	}{
		{"equals formula", "=HYPERLINK(\"http://evil\",\"x\")", "'=HYPERLINK(\"http://evil\",\"x\")"},
		{"plus formula", "+cmd", "'+cmd"},
		{"minus formula", "-2+3", "'-2+3"},
		{"at formula", "@SUM(A1:A2)", "'@SUM(A1:A2)"},
		{"leading whitespace formula", "  =SUM(A1)", "'  =SUM(A1)"},
		{"plain text unchanged", "Ana Maria", "Ana Maria"},
		{"number unchanged", "573001234567", "573001234567"},
		{"empty unchanged", "", ""},
		{"quote only whitespace", "   ", "   "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := csvSanitizeCell(tc.in); got != tc.expected {
				t.Errorf("csvSanitizeCell(%q) = %q, want %q", tc.in, got, tc.expected)
			}
		})
	}
}

func TestExportServiceStreamWritesBOMHeaderAndRows(t *testing.T) {
	svc := NewExportService()
	var buf bytes.Buffer

	headers := []string{"ID", "Nombre"}
	pages := 0
	written, err := svc.Stream(context.Background(), &buf, headers, func(_ context.Context, offset int32) ([][]string, error) {
		pages++
		if offset == 0 {
			return [][]string{{"1", "Ana"}, {"2", "Luis"}}, nil
		}
		return [][]string{}, nil
	})
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}
	if written != 2 {
		t.Errorf("expected 2 data rows written, got %d", written)
	}

	out := buf.String()
	if !bytes.HasPrefix(buf.Bytes(), utf8BOM) {
		t.Errorf("output must start with UTF-8 BOM, got prefix %q", buf.Bytes()[:3])
	}

	r := csv.NewReader(strings.NewReader(out))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("output is not valid CSV: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected 3 records (header + 2 rows), got %d", len(records))
	}
	if records[0][1] != "Nombre" {
		t.Errorf("header row mismatch: %v", records[0])
	}
	if records[1][1] != "Ana" {
		t.Errorf("row 1 mismatch: %v", records[1])
	}
	if pages != 1 {
		t.Errorf("expected 1 page call (short page stops loop), got %d", pages)
	}
}

func TestExportServiceStreamPaginatesUntilEmpty(t *testing.T) {
	svc := NewExportService()
	var buf bytes.Buffer

	// 3 pages of ExportPageSize then a short final page.
	callCount := 0
	written, err := svc.Stream(context.Background(), &buf, []string{"v"}, func(_ context.Context, offset int32) ([][]string, error) {
		callCount++
		if callCount <= 2 {
			rows := make([][]string, ExportPageSize)
			for i := range rows {
				rows[i] = []string{"x"}
			}
			return rows, nil
		}
		return [][]string{{"x"}}, nil
	})
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 page calls, got %d", callCount)
	}
	if written != 2*ExportPageSize+1 {
		t.Errorf("expected %d written rows, got %d", 2*ExportPageSize+1, written)
	}

	r := csv.NewReader(strings.NewReader(buf.String()))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("invalid CSV: %v", err)
	}
	// header + 2*ExportPageSize + 1 final row
	if want := 1 + 2*ExportPageSize + 1; len(records) != want {
		t.Errorf("expected %d records, got %d", want, len(records))
	}
}

func TestExportServiceStreamSanitizesEveryCell(t *testing.T) {
	svc := NewExportService()
	var buf bytes.Buffer
	_, err := svc.Stream(context.Background(), &buf, []string{"v"}, func(_ context.Context, _ int32) ([][]string, error) {
		return [][]string{{"=1+1"}, {"@foo"}}, nil
	})
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}
	if !strings.Contains(buf.String(), "'=1+1") {
		t.Errorf("formula cell not sanitized: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "'@foo") {
		t.Errorf("at-cell not sanitized: %s", buf.String())
	}
}

func TestMapContactoCSVMasksWithdrawnConsent(t *testing.T) {
	now := time.Now()
	companyID := int32(7)
	contacts := []*domain.Contact{
		{
			ID: 1, PhoneNumber: "573001234567", DisplayName: "Ana",
			Email: "ana@correo.co", TipoDocumento: domain.TipoDocCC,
			NumeroDocumento: "123", CompanyID: &companyID,
			Source: domain.ContactSourceManual, LeadStatus: domain.LeadStatusCliente,
			ConsentStatus: domain.ConsentStatusWithdrawn, CreatedAt: now,
		},
		{
			ID: 2, PhoneNumber: "573009999999", DisplayName: "Luis",
			Email: "luis@correo.co", TipoDocumento: domain.TipoDocNIT,
			NumeroDocumento: "456",
			Source:          domain.ContactSourceWhatsApp, LeadStatus: domain.LeadStatusNuevo,
			ConsentStatus: domain.ConsentStatusGranted, CreatedAt: now,
		},
	}

	rows := MapContactoCSV(contacts)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	masked := rows[0]
	if masked[2] != maskTelefono || masked[1] != maskNombre || masked[3] != maskEmail {
		t.Errorf("withdrawn contact PII not masked: %v", masked)
	}
	if masked[4] != maskDocumento || masked[5] != maskDocumento {
		t.Errorf("withdrawn contact document not masked: %v", masked)
	}
	if masked[6] != "7" {
		t.Errorf("non-PII company id should remain: %v", masked)
	}

	clear := rows[1]
	if clear[2] != "573009999999" || clear[1] != "Luis" || clear[3] != "luis@correo.co" {
		t.Errorf("granted contact PII should be real: %v", clear)
	}
}

func TestMapContactoCSVHeaders(t *testing.T) {
	if len(ContactoCSVHeaders) != len(contactExampleRow()) {
		t.Errorf("contact header count %d does not match row width %d", len(ContactoCSVHeaders), len(contactExampleRow()))
	}
}

func contactExampleRow() []string {
	return []string{"", "", "", "", "", "", "", "", "", "", ""}
}
