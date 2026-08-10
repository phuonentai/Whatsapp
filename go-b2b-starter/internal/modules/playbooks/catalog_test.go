package playbooks

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moasq/go-b2b-starter/internal/modules/playbooks/app/services"
)

func TestCatalogSeedsFiveVerticalPlaybooks(t *testing.T) {
	catalog := Catalog()
	require.Len(t, catalog, 5)

	keys := make([]string, 0, len(catalog))
	for _, pb := range catalog {
		keys = append(keys, pb.Key)
	}
	assert.ElementsMatch(t, []string{"comercio", "restaurantes", "citas", "servicios-profesionales", "talleres"}, keys)
}

func TestCatalogPlaybooksAreCompleteAndValid(t *testing.T) {
	for _, pb := range Catalog() {
		t.Run(pb.Key, func(t *testing.T) {
			assert.NotEmpty(t, pb.Name, "name required")
			assert.NotEmpty(t, pb.Vertical, "vertical required")
			assert.NotEmpty(t, pb.Description, "description required")
			assert.Empty(t, pb.RequiresModules, "no playbook requires a module today")

			payload, err := services.ParsePayload(pb.Payload)
			require.NoError(t, err, "payload must be structurally valid")

			assert.NotEmpty(t, payload.Pipeline.Nombre)
			assert.GreaterOrEqual(t, len(payload.Pipeline.Etapas), 1, "at least one stage")
			for _, etapa := range payload.Pipeline.Etapas {
				assert.NotEmpty(t, etapa.Nombre)
				assert.GreaterOrEqual(t, etapa.Orden, int32(1))
				assert.NotEmpty(t, etapa.Color)
			}
			assert.GreaterOrEqual(t, len(payload.Tags), 1, "at least one tag")
			assert.GreaterOrEqual(t, len(payload.Guiones), 1, "at least one guion")
			sequenceCount := 0
			for _, guion := range payload.Guiones {
				assert.NotEmpty(t, guion.ID)
				assert.NotEmpty(t, guion.Titulo)
				if len(guion.Pasos) > 0 {
					sequenceCount++
					assert.Empty(t, guion.Mensaje, "sequence guion must not carry a single-shot mensaje")
					assert.GreaterOrEqual(t, len(guion.Pasos), 2, "sequence must have 2+ steps")
					for _, paso := range guion.Pasos {
						assert.NotEmpty(t, paso.ID)
						assert.NotEmpty(t, paso.Titulo)
						assert.NotEmpty(t, paso.Mensaje)
					}
					continue
				}
				assert.NotEmpty(t, guion.Mensaje, "single-shot guion requires mensaje")
			}
			assert.GreaterOrEqual(t, sequenceCount, 1, "each vertical ships at least one scripted sequence")
			for moduleKey, preset := range payload.ModuleConfigs {
				assert.Equal(t, "tickets", moduleKey, "only the shipped tickets module may be referenced")
				sla, ok := preset["sla_hours"].(map[string]any)
				require.True(t, ok, "sla_hours must be an object")
				for priority, hours := range sla {
					switch hours.(type) {
					case float64, int, int64:
					default:
						t.Errorf("sla_hours[%s] must be numeric, got %T", priority, hours)
					}
				}
				for _, key := range []string{"priorities", "tags"} {
					arr, ok := preset[key].([]any)
					require.True(t, ok, "%s must be a string list", key)
					require.NotEmpty(t, arr)
					for _, item := range arr {
						_, ok := item.(string)
						require.True(t, ok, "%s items must be strings", key)
					}
				}
			}
		})
	}
}
