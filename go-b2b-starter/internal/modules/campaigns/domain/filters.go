package domain

import (
	"fmt"
	"strings"
)

// Whitelisted filter fields and their operators. Any field/op outside these
// tables is rejected — filter_spec can never express SQL or bypass hard gates
// (consent, valid phone) that the evaluator appends unconditionally.
const (
	FieldSource      = "source"
	FieldLeadStatus  = "lead_status"
	FieldCompanyID   = "company_id"
	FieldAssignedTo  = "assigned_to"
	FieldTagIDs      = "tag_ids"
	FieldRecencyDays = "recency_days"
	FieldSearch      = "search"

	OpEq       = "eq"
	OpAny      = "any"
	OpLte      = "lte"
	OpContains = "contains"
)

// validLeadStatuses are the CRM contact lead statuses (crm.contacts CHECK).
var validLeadStatuses = map[string]bool{
	"nuevo": true, "contactado": true, "calificado": true,
	"descalificado": true, "cliente": true,
}

type fieldSpec struct {
	ops       map[string]bool
	valueType string // "string" | "int" | "int_array"
}

var whitelist = map[string]fieldSpec{
	FieldSource:      {ops: map[string]bool{OpEq: true}, valueType: "string"},
	FieldLeadStatus:  {ops: map[string]bool{OpEq: true}, valueType: "string"},
	FieldCompanyID:   {ops: map[string]bool{OpEq: true}, valueType: "int"},
	FieldAssignedTo:  {ops: map[string]bool{OpEq: true}, valueType: "int"},
	FieldTagIDs:      {ops: map[string]bool{OpAny: true}, valueType: "int_array"},
	FieldRecencyDays: {ops: map[string]bool{OpLte: true}, valueType: "int"},
	FieldSearch:      {ops: map[string]bool{OpContains: true}, valueType: "string"},
}

// ValidateFilterSpec validates a filter spec against the whitelist.
// It returns ErrInvalidFilterSpec wrapping a Spanish message on the first
// violation. Used by manual CRUD and the AI audience builder alike.
func ValidateFilterSpec(spec []Filter) error {
	if len(spec) == 0 {
		return fmt.Errorf("%w: agrega al menos un filtro", ErrInvalidFilterSpec)
	}
	for i, f := range spec {
		fs, ok := whitelist[f.Field]
		if !ok {
			return fmt.Errorf("%w: campo '%s' no permitido", ErrInvalidFilterSpec, f.Field)
		}
		if !fs.ops[f.Op] {
			return fmt.Errorf("%w: operador '%s' no permitido para el campo '%s'", ErrInvalidFilterSpec, f.Op, f.Field)
		}
		if err := validateValue(f.Field, f.Op, f.Value); err != nil {
			return fmt.Errorf("%w: filtro %d (%s): %v", ErrInvalidFilterSpec, i+1, f.Field, err)
		}
	}
	return nil
}

func validateValue(field, op string, value any) error {
	switch whitelist[field].valueType {
	case "string":
		s, ok := value.(string)
		if !ok || strings.TrimSpace(s) == "" {
			return fmt.Errorf("el valor debe ser texto no vacío")
		}
		if field == FieldLeadStatus && !validLeadStatuses[s] {
			return fmt.Errorf("estado comercial inválido: %s (permitidos: nuevo, contactado, calificado, descalificado, cliente)", s)
		}
		return nil
	case "int":
		n, ok := toInt(value)
		if !ok || n <= 0 {
			return fmt.Errorf("el valor debe ser un número entero positivo")
		}
		return nil
	case "int_array":
		arr, ok := toIntArray(value)
		if !ok || len(arr) == 0 {
			return fmt.Errorf("el valor debe ser una lista de IDs de etiquetas")
		}
		return nil
	}
	return fmt.Errorf("tipo de valor desconocido")
}

func toInt(value any) (int32, bool) {
	switch v := value.(type) {
	case float64:
		if v != float64(int32(v)) {
			return 0, false
		}
		return int32(v), true
	case int:
		return int32(v), true
	case int32:
		return v, true
	case int64:
		return int32(v), true
	}
	return 0, false
}

func toIntArray(value any) ([]int32, bool) {
	arr, ok := value.([]any)
	if !ok {
		// Accept a single number as a one-element array.
		if n, ok := toInt(value); ok {
			return []int32{n}, true
		}
		return nil, false
	}
	out := make([]int32, 0, len(arr))
	for _, item := range arr {
		n, ok := toInt(item)
		if !ok || n <= 0 {
			return nil, false
		}
		out = append(out, n)
	}
	return out, true
}
