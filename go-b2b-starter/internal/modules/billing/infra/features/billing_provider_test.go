package features

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseModuleKeys_TopLevel(t *testing.T) {
	keys := parseModuleKeys(map[string]any{
		"module_tickets": "true",
		"crm_features":   "crm_contacts_manage",
	})
	assert.ElementsMatch(t, []string{"tickets"}, keys)
}

func TestParseModuleKeys_NestedProductMetadata(t *testing.T) {
	keys := parseModuleKeys(map[string]any{
		"product_metadata": map[string]any{
			"module_tickets": "true",
			"module_agents":  "true",
		},
	})
	assert.ElementsMatch(t, []string{"tickets", "agents"}, keys)
}

func TestParseModuleKeys_NestedStringMap(t *testing.T) {
	keys := parseModuleKeys(map[string]any{
		"product_metadata": map[string]string{"module_tickets": "true"},
	})
	assert.ElementsMatch(t, []string{"tickets"}, keys)
}

func TestParseModuleKeys_None(t *testing.T) {
	keys := parseModuleKeys(map[string]any{"crm_features": "x"})
	assert.Empty(t, keys)
}

func TestParseCRMFeatures_DisabledSubscription(t *testing.T) {
	features := parseCRMFeatures(map[string]any{}, false)
	assert.False(t, features["crm_deals"])
	assert.False(t, features["crm_tags"])
}

func TestParseCRMFeatures_TopLevelAndNested(t *testing.T) {
	features := parseCRMFeatures(map[string]any{
		"crm_features": "crm_deals,crm_companies",
	}, true)
	assert.True(t, features["crm_deals"])
	assert.True(t, features["crm_companies"])

	nested := parseCRMFeatures(map[string]any{
		"product_metadata": map[string]any{"crm_features": "crm_activities"},
	}, true)
	assert.True(t, nested["crm_activities"])
}

func TestParseAIFeatures_DisabledSubscription(t *testing.T) {
	features := parseAIFeatures(make(map[string]bool), map[string]any{}, false)
	assert.False(t, features["ai_assistant"])
}

func TestParseAIFeatures_DirectKeyAndList(t *testing.T) {
	features := parseAIFeatures(make(map[string]bool), map[string]any{
		"ai_assistant": true,
		"ai_features":  "ai_smart_replies,ai_summarization",
	}, true)
	assert.True(t, features["ai_assistant"])
	assert.True(t, features["ai_smart_replies"])
	assert.True(t, features["ai_summarization"])
}

func TestParseAIFeatures_StringBoolValue(t *testing.T) {
	features := parseAIFeatures(make(map[string]bool), map[string]any{
		"ai_assistant": "true",
	}, true)
	assert.True(t, features["ai_assistant"])

	off := parseAIFeatures(make(map[string]bool), map[string]any{
		"ai_assistant": "false",
	}, true)
	assert.False(t, off["ai_assistant"])
}

// conversation-row-scoping (Decisión 8): el flag es grant base solo para
// suscripciones activas/trialing/past_due; free/inactiva → false.
func TestBasePaidPlanFeatures_ActiveGrantsScoping(t *testing.T) {
	features := basePaidPlanFeatures(true)
	assert.True(t, features["conversation_row_scoping"])
}

func TestBasePaidPlanFeatures_InactiveOrFreeNoScoping(t *testing.T) {
	features := basePaidPlanFeatures(false)
	assert.False(t, features["conversation_row_scoping"])
}

func TestParseQuotas_IncludesAiCredits(t *testing.T) {
	quotas := parseQuotas(map[string]any{
		"ai_credits_max": "1000",
		"max_contactos":  "500",
	})
	assert.Equal(t, int32(1000), quotas["ai_credits"])
	assert.Equal(t, int32(500), quotas["contacts"])
}

func TestParseModuleKeys_DefaultGrantedModulesAlwaysIncluded(t *testing.T) {
	keys := parseModuleKeys(map[string]any{"crm_features": "x"})
	assert.Empty(t, keys)
	keys = append(keys, defaultGrantedModules...)
	assert.Contains(t, keys, "analytics")
}
