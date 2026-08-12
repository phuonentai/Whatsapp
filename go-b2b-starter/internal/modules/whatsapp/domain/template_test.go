package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func templateIn(status TemplateStatus) *Template {
	return &Template{Status: status}
}

func TestTemplateStateMachine_AllowedTransitions(t *testing.T) {
	cases := []struct {
		from   TemplateStatus
		to     TemplateStatus
		expect bool
	}{
		{TemplateStatusDraft, TemplateStatusSubmitted, true},
		{TemplateStatusSubmitted, TemplateStatusApproved, true},
		{TemplateStatusSubmitted, TemplateStatusRejected, true},
		{TemplateStatusSubmitted, TemplateStatusPaused, true},
		{TemplateStatusRejected, TemplateStatusDraft, true},
		{TemplateStatusPaused, TemplateStatusSubmitted, true},
		// Forbidden transitions.
		{TemplateStatusDraft, TemplateStatusApproved, false},
		{TemplateStatusDraft, TemplateStatusRejected, false},
		{TemplateStatusDraft, TemplateStatusDraft, false},
		{TemplateStatusApproved, TemplateStatusDraft, false},
		{TemplateStatusApproved, TemplateStatusRejected, false},
		{TemplateStatusApproved, TemplateStatusSubmitted, false},
		{TemplateStatusRejected, TemplateStatusSubmitted, false},
		{TemplateStatusRejected, TemplateStatusApproved, false},
		{TemplateStatusPaused, TemplateStatusApproved, false},
		{TemplateStatusPaused, TemplateStatusDraft, false},
		{TemplateStatusPaused, TemplateStatusRejected, false},
	}
	for _, tc := range cases {
		t.Run(string(tc.from)+"→"+string(tc.to), func(t *testing.T) {
			tmpl := templateIn(tc.from)
			assert.Equal(t, tc.expect, tmpl.CanTransitionTo(tc.to))
			err := tmpl.Transition(tc.to)
			if tc.expect {
				require.NoError(t, err)
				assert.Equal(t, tc.to, tmpl.Status)
			} else {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrTemplateInvalidTransition)
				assert.Equal(t, tc.from, tmpl.Status, "forbidden transition must not change status")
			}
		})
	}
}

func TestCountParams(t *testing.T) {
	cases := []struct {
		body string
		want int
	}{
		{"", 0},
		{"Hola", 0},
		{"Hola {{1}}", 1},
		{"Hola {{1}}, tu pedido {{2}} fue confirmado.", 2},
		{"Hola {{2}}, tu pedido {{1}} fue confirmado.", 2}, // max index
		{"Hola {{ 1 }} y {{2}}", 2},                         // whitespace tolerated
		{"Hola {{1}} {{1}}", 1},                             // duplicate param
		{"Hola {1}", 0},                                     // single braces ignored
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, CountParams(tc.body), "body %q", tc.body)
	}
}

func TestTemplateValidate(t *testing.T) {
	valid := &Template{Name: "n", Category: "UTILITY", Language: "es", Body: "Hola {{1}}"}
	assert.NoError(t, valid.Validate())

	assert.ErrorContains(t, (&Template{Category: "UTILITY", Language: "es", Body: "x"}).Validate(), "El nombre de la plantilla es obligatorio")
	assert.ErrorContains(t, (&Template{Name: "n", Language: "es", Body: "x"}).Validate(), "La categoría de la plantilla es obligatoria")
	assert.ErrorContains(t, (&Template{Name: "n", Category: "UTILITY", Body: "x"}).Validate(), "El idioma de la plantilla es obligatorio")
	assert.ErrorContains(t, (&Template{Name: "n", Category: "UTILITY", Language: "es"}).Validate(), "El cuerpo de la plantilla es obligatorio")
}

func TestTemplateIsEditable(t *testing.T) {
	assert.True(t, templateIn(TemplateStatusDraft).IsEditable())
	assert.False(t, templateIn(TemplateStatusSubmitted).IsEditable())
	assert.False(t, templateIn(TemplateStatusApproved).IsEditable())
	assert.False(t, templateIn(TemplateStatusRejected).IsEditable())
	assert.False(t, templateIn(TemplateStatusPaused).IsEditable())
}
