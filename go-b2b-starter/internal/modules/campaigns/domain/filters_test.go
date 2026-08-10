package domain

import (
	"strings"
	"testing"
)

func TestValidateFilterSpec(t *testing.T) {
	cases := []struct {
		name string
		spec []Filter
		want bool
	}{
		{
			name: "all whitelisted fields valid",
			spec: []Filter{
				{Field: FieldSource, Op: OpEq, Value: "whatsapp"},
				{Field: FieldLeadStatus, Op: OpEq, Value: "cliente"},
				{Field: FieldCompanyID, Op: OpEq, Value: float64(5)},
				{Field: FieldAssignedTo, Op: OpEq, Value: float64(9)},
				{Field: FieldTagIDs, Op: OpAny, Value: []any{float64(3), float64(7)}},
				{Field: FieldRecencyDays, Op: OpLte, Value: float64(30)},
				{Field: FieldSearch, Op: OpContains, Value: "Bogotá"},
			},
			want: true,
		},
		{name: "empty spec rejected", spec: nil, want: false},
		{name: "unknown field rejected", spec: []Filter{{Field: "password", Op: OpEq, Value: "x"}}, want: false},
		{name: "unknown op rejected", spec: []Filter{{Field: FieldSource, Op: "gt", Value: "whatsapp"}}, want: false},
		{name: "op not allowed for field", spec: []Filter{{Field: FieldSource, Op: OpAny, Value: []any{1}}}, want: false},
		{name: "empty string value rejected", spec: []Filter{{Field: FieldSource, Op: OpEq, Value: ""}}, want: false},
		{name: "non-numeric int field rejected", spec: []Filter{{Field: FieldCompanyID, Op: OpEq, Value: "abc"}}, want: false},
		{name: "zero company id rejected", spec: []Filter{{Field: FieldCompanyID, Op: OpEq, Value: float64(0)}}, want: false},
		{name: "invalid lead status rejected", spec: []Filter{{Field: FieldLeadStatus, Op: OpEq, Value: "vip"}}, want: false},
		{name: "empty tag list rejected", spec: []Filter{{Field: FieldTagIDs, Op: OpAny, Value: []any{}}}, want: false},
		{name: "invalid tag item rejected", spec: []Filter{{Field: FieldTagIDs, Op: OpAny, Value: []any{"a"}}}, want: false},
		{name: "float non-integer rejected", spec: []Filter{{Field: FieldRecencyDays, Op: OpLte, Value: float64(2.5)}}, want: false},
		{
			name: "duplicate fields allowed (AND)",
			spec: []Filter{
				{Field: FieldLeadStatus, Op: OpEq, Value: "cliente"},
				{Field: FieldLeadStatus, Op: OpEq, Value: "calificado"},
			},
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateFilterSpec(tc.spec)
			if tc.want && err != nil {
				t.Fatalf("expected valid spec, got: %v", err)
			}
			if !tc.want && err == nil {
				t.Fatalf("expected invalid spec to be rejected")
			}
			if !tc.want && err != nil && !strings.Contains(err.Error(), "no permitido") && !strings.Contains(err.Error(), "inválido") && !strings.Contains(err.Error(), "obligatorio") && !strings.Contains(err.Error(), "no existe") && !strings.Contains(err.Error(), "debe ser") && !strings.Contains(err.Error(), "filtro") {
				t.Fatalf("expected Spanish error message, got: %v", err)
			}
		})
	}
}
