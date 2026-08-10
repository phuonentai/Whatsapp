package services

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// ExportPageSize is the number of rows fetched per page while streaming an
// export. Kept modest so large exports stream incrementally instead of
// buffering the full dataset in memory.
const ExportPageSize = 500

// utf8BOM is the UTF-8 byte order mark prefixed to every export so Spanish
// accents render correctly in Microsoft Excel on Windows.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// PageFunc fetches the next page of pre-mapped rows starting at offset.
// It returns the rows of the page; the export loop stops when an empty page
// (or a page shorter than ExportPageSize) is returned.
type PageFunc func(ctx context.Context, offset int32) ([][]string, error)

// ExportService streams entities as CSV through a pagination callback.
type ExportService struct{}

func NewExportService() *ExportService {
	return &ExportService{}
}

// Stream writes the UTF-8 BOM, the header row, then each page of rows via the
// pagination callback, sanitizing every cell against formula injection. It
// returns the number of data rows written (excludes the header row).
func (s *ExportService) Stream(ctx context.Context, w io.Writer, headers []string, page PageFunc) (int, error) {
	if _, err := w.Write(utf8BOM); err != nil {
		return 0, fmt.Errorf("failed to write UTF-8 BOM: %w", err)
	}

	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(headers); err != nil {
		return 0, fmt.Errorf("failed to write CSV headers: %w", err)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return 0, fmt.Errorf("failed to flush CSV headers: %w", err)
	}

	var written int
	var offset int32
	for {
		rows, err := page(ctx, offset)
		if err != nil {
			return written, fmt.Errorf("failed to fetch CSV page: %w", err)
		}
		if len(rows) == 0 {
			return written, nil
		}

		for _, row := range rows {
			sanitized := make([]string, len(row))
			for i, cell := range row {
				sanitized[i] = csvSanitizeCell(cell)
			}
			if err := writer.Write(sanitized); err != nil {
				return written, fmt.Errorf("failed to write CSV row: %w", err)
			}
			written++
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return written, fmt.Errorf("failed to flush CSV rows: %w", err)
		}

		if len(rows) < ExportPageSize {
			return written, nil
		}
		offset += int32(len(rows))
	}
}

// csvSanitizeCell neutralizes spreadsheet formula injection: when the trimmed
// value starts with =, +, -, or @, a single quote is prefixed so Excel renders
// the cell as literal text instead of executing it as a formula.
func csvSanitizeCell(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed == "" {
		return value
	}
	switch trimmed[0] {
	case '=', '+', '-', '@':
		return "'" + value
	default:
		return value
	}
}

// ImportSummary is the response of a CSV import: counts plus per-row errors.
type ImportSummary struct {
	Importados int32         `json:"importados"`
	Omitidos   int32         `json:"omitidos"`
	Errores    []ImportError `json:"errores"`
}

// ImportError reports a single rejected row.
type ImportError struct {
	Fila  int32  `json:"fila"`
	Razon string `json:"razon"`
}
