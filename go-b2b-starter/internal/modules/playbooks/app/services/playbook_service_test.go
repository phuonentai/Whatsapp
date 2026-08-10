package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moasq/go-b2b-starter/internal/modules/playbooks/domain"
	registryServices "github.com/moasq/go-b2b-starter/internal/modules/registry/app/services"
	registryDomain "github.com/moasq/go-b2b-starter/internal/modules/registry/domain"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// ---- fakes ----

type nilLogger struct{}

func (nilLogger) Debug(string, ...loggerDomain.Fields)               {}
func (nilLogger) Info(string, ...loggerDomain.Fields)                {}
func (nilLogger) Warn(string, ...loggerDomain.Fields)                {}
func (nilLogger) Error(string, ...loggerDomain.Fields)               {}
func (nilLogger) Fatal(string, ...loggerDomain.Fields)               {}
func (nilLogger) WithFields(loggerDomain.Fields) loggerDomain.Logger { return nilLogger{} }

type fakePlaybookRepo struct {
	playbooks map[string]*domain.Playbook
}

func (f *fakePlaybookRepo) ListActive(ctx context.Context) ([]*domain.Playbook, error) {
	out := make([]*domain.Playbook, 0, len(f.playbooks))
	for _, pb := range f.playbooks {
		out = append(out, pb)
	}
	return out, nil
}

func (f *fakePlaybookRepo) GetByKey(ctx context.Context, key string) (*domain.Playbook, error) {
	pb, ok := f.playbooks[key]
	if !ok {
		return nil, domain.ErrPlaybookNotFound
	}
	return pb, nil
}

type fakeOrgPlaybookRepo struct {
	states map[string]*domain.OrganizationPlaybook
}

func (f *fakeOrgPlaybookRepo) ListByOrg(ctx context.Context, orgID int32) ([]*domain.OrganizationPlaybook, error) {
	out := make([]*domain.OrganizationPlaybook, 0, len(f.states))
	for _, s := range f.states {
		out = append(out, s)
	}
	return out, nil
}
func (f *fakeOrgPlaybookRepo) GetByOrgKey(ctx context.Context, orgID int32, key string) (*domain.OrganizationPlaybook, error) {
	return f.states[key], nil
}
func (f *fakeOrgPlaybookRepo) Upsert(ctx context.Context, orgID int32, key string, seededPipelineID *int32) (*domain.OrganizationPlaybook, error) {
	s := &domain.OrganizationPlaybook{OrganizationID: orgID, PlaybookKey: key, SeededPipelineID: seededPipelineID, AppliedAt: "2026-01-01"}
	f.states[key] = s
	return s, nil
}
func (f *fakeOrgPlaybookRepo) Delete(ctx context.Context, orgID int32, key string) error {
	delete(f.states, key)
	return nil
}

type fakeApplyRepo struct {
	applied map[string]int
	lastPb  *domain.Playbook
	lastPld *domain.PlaybookPayload
	err     error
}

func (f *fakeApplyRepo) Apply(ctx context.Context, orgID int32, pb *domain.Playbook, payload *domain.PlaybookPayload) (*domain.OrganizationPlaybook, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.applied[pb.Key]++
	f.lastPb = pb
	f.lastPld = payload
	return &domain.OrganizationPlaybook{OrganizationID: orgID, PlaybookKey: pb.Key, AppliedAt: "2026-01-01"}, nil
}
func (f *fakeApplyRepo) Reset(ctx context.Context, orgID int32, pb *domain.Playbook, payload *domain.PlaybookPayload) error {
	if f.err != nil {
		return f.err
	}
	delete(f.applied, pb.Key)
	return nil
}

type fakeModuleSvc struct {
	orgMods []*registryDomain.OrganizationModule
}

func (f *fakeModuleSvc) ListCatalog(ctx context.Context) ([]*registryDomain.Module, error) {
	return nil, nil
}
func (f *fakeModuleSvc) ListCatalogInternal(ctx context.Context) ([]*registryDomain.Module, error) {
	return nil, nil
}
func (f *fakeModuleSvc) ListOrgModules(ctx context.Context, orgID int32) ([]*registryDomain.Module, []*registryDomain.OrganizationModule, error) {
	return nil, f.orgMods, nil
}
func (f *fakeModuleSvc) SaveOrgModuleConfig(ctx context.Context, orgID int32, moduleKey string, config map[string]any) (*registryDomain.OrganizationModule, error) {
	return nil, nil
}
func (f *fakeModuleSvc) SyncModulesFromMetadata(ctx context.Context, orgID int32, moduleKeys []string) error {
	return nil
}
func (f *fakeModuleSvc) ResolveGrantedFeatures(ctx context.Context, enabledKeys []string) map[string]bool {
	return nil
}
func (f *fakeModuleSvc) ValidateConfig(ctx context.Context, moduleKey string, config map[string]any) error {
	if moduleKey == "tickets" {
		if raw, ok := config["sla_hours"]; ok {
			if _, ok := raw.(map[string]any); !ok {
				return errors.New("sla_hours must be an object")
			}
		}
		return nil
	}
	return registryDomain.ErrModuleNotFound
}

func testPlaybookPayload(t *testing.T, presets map[string]map[string]any) *domain.PlaybookPayload {
	t.Helper()
	return &domain.PlaybookPayload{
		Pipeline: domain.PipelineTemplate{
			Nombre: "Pipeline Test",
			Etapas: []domain.EtapaTemplate{{Nombre: "Nuevo", Orden: 1, Color: "#000", Probabilidad: 10}},
		},
		Tags:          []string{"tag-a"},
		ModuleConfigs: presets,
		Guiones:       []domain.Guion{{ID: "g1", Titulo: "Saludo", Mensaje: "Hola"}},
	}
}

func testPlaybook(t *testing.T, payload *domain.PlaybookPayload, requires []string) *domain.Playbook {
	t.Helper()
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	return &domain.Playbook{Key: "comercio", Name: "Comercio", Vertical: "retail", RequiresModules: requires, Payload: raw, IsActive: true}
}

func newTestService(pb *domain.Playbook, moduleSvc registryServices.ModuleService) (PlaybookService, *fakeOrgPlaybookRepo, *fakeApplyRepo) {
	pbRepo := &fakePlaybookRepo{playbooks: map[string]*domain.Playbook{pb.Key: pb}}
	orgRepo := &fakeOrgPlaybookRepo{states: map[string]*domain.OrganizationPlaybook{}}
	applyRepo := &fakeApplyRepo{applied: map[string]int{}}
	return NewPlaybookService(pbRepo, orgRepo, applyRepo, moduleSvc, nilLogger{}), orgRepo, applyRepo
}

// ---- tests ----

func TestApplySeedsPlaybook(t *testing.T) {
	payload := testPlaybookPayload(t, map[string]map[string]any{
		"tickets": {"sla_hours": map[string]any{"low": 48}, "priorities": []any{"low", "normal", "high"}},
	})
	pb := testPlaybook(t, payload, nil)
	svc, _, applyRepo := newTestService(pb, &fakeModuleSvc{orgMods: []*registryDomain.OrganizationModule{
		{ModuleKey: "tickets"},
	}})

	state, err := svc.Apply(context.Background(), 1, "comercio")
	require.NoError(t, err)
	assert.Equal(t, "comercio", state.PlaybookKey)
	assert.Equal(t, 1, applyRepo.applied["comercio"])
}

func TestApplyRejectsUnknownPlaybook(t *testing.T) {
	payload := testPlaybookPayload(t, nil)
	pb := testPlaybook(t, payload, nil)
	svc, _, _ := newTestService(pb, &fakeModuleSvc{})

	_, err := svc.Apply(context.Background(), 1, "nope")
	require.ErrorIs(t, err, domain.ErrPlaybookNotFound)
}

func TestApplyRejectsInvalidPayload(t *testing.T) {
	raw := []byte(`{"pipeline":{"nombre":"","etapas":[]},"tags":[],"guiones":[]}`)
	pb := &domain.Playbook{Key: "comercio", Payload: raw}
	svc, _, _ := newTestService(pb, &fakeModuleSvc{})

	_, err := svc.Apply(context.Background(), 1, "comercio")
	require.ErrorIs(t, err, domain.ErrInvalidPlaybookPayload)
}

func TestApplyAcceptsSequenceGuion(t *testing.T) {
	payload := testPlaybookPayload(t, nil)
	payload.Guiones = []domain.Guion{{
		ID: "g1", Titulo: "Confirmar pedido",
		Pasos: []domain.GuionPaso{
			{ID: "p1", Titulo: "Detalle", Mensaje: "¿Qué quieres?"},
			{ID: "p2", Titulo: "Dirección", Mensaje: "¿A qué dirección?"},
		},
	}}
	pb := testPlaybook(t, payload, nil)
	svc, _, applyRepo := newTestService(pb, &fakeModuleSvc{})

	_, err := svc.Apply(context.Background(), 1, "comercio")
	require.NoError(t, err)
	assert.Equal(t, 1, applyRepo.applied["comercio"])
}

func TestApplyRejectsSequenceWithSingleStep(t *testing.T) {
	payload := testPlaybookPayload(t, nil)
	payload.Guiones = []domain.Guion{{
		ID: "g1", Titulo: "Confirmar pedido",
		Pasos: []domain.GuionPaso{{ID: "p1", Titulo: "Único", Mensaje: "Solo un paso"}},
	}}
	pb := testPlaybook(t, payload, nil)
	svc, _, applyRepo := newTestService(pb, &fakeModuleSvc{})

	_, err := svc.Apply(context.Background(), 1, "comercio")
	require.ErrorIs(t, err, domain.ErrInvalidPlaybookPayload)
	assert.Zero(t, applyRepo.applied["comercio"])
}

func TestApplyRejectsIncompleteSequenceStep(t *testing.T) {
	payload := testPlaybookPayload(t, nil)
	payload.Guiones = []domain.Guion{{
		ID: "g1", Titulo: "Confirmar pedido",
		Pasos: []domain.GuionPaso{
			{ID: "p1", Titulo: "Detalle", Mensaje: "¿Qué quieres?"},
			{ID: "p2", Titulo: "", Mensaje: "Falta titulo"},
		},
	}}
	pb := testPlaybook(t, payload, nil)
	svc, _, applyRepo := newTestService(pb, &fakeModuleSvc{})

	_, err := svc.Apply(context.Background(), 1, "comercio")
	require.ErrorIs(t, err, domain.ErrInvalidPlaybookPayload)
	assert.Zero(t, applyRepo.applied["comercio"])
}

func TestApplyRejectsGuionWithMensajeAndPasos(t *testing.T) {
	payload := testPlaybookPayload(t, nil)
	payload.Guiones = []domain.Guion{{
		ID: "g1", Titulo: "Confirmar pedido", Mensaje: "Hola",
		Pasos: []domain.GuionPaso{
			{ID: "p1", Titulo: "Detalle", Mensaje: "¿Qué quieres?"},
			{ID: "p2", Titulo: "Dirección", Mensaje: "¿A qué dirección?"},
		},
	}}
	pb := testPlaybook(t, payload, nil)
	svc, _, applyRepo := newTestService(pb, &fakeModuleSvc{})

	_, err := svc.Apply(context.Background(), 1, "comercio")
	require.ErrorIs(t, err, domain.ErrInvalidPlaybookPayload)
	assert.Zero(t, applyRepo.applied["comercio"])
}

func TestApplyRejectsGuionWithoutMensajeOrPasos(t *testing.T) {
	payload := testPlaybookPayload(t, nil)
	payload.Guiones = []domain.Guion{{ID: "g1", Titulo: "Vacío"}}
	pb := testPlaybook(t, payload, nil)
	svc, _, applyRepo := newTestService(pb, &fakeModuleSvc{})

	_, err := svc.Apply(context.Background(), 1, "comercio")
	require.ErrorIs(t, err, domain.ErrInvalidPlaybookPayload)
	assert.Zero(t, applyRepo.applied["comercio"])
}

func TestApplyRejectsInvalidModuleConfigPreset(t *testing.T) {
	payload := testPlaybookPayload(t, map[string]map[string]any{
		"tickets": {"sla_hours": "no-un-numero"},
	})
	pb := testPlaybook(t, payload, nil)
	svc, _, applyRepo := newTestService(pb, &fakeModuleSvc{orgMods: []*registryDomain.OrganizationModule{
		{ModuleKey: "tickets"},
	}})

	_, err := svc.Apply(context.Background(), 1, "comercio")
	require.ErrorIs(t, err, domain.ErrInvalidModuleConfigPreset)
	assert.Zero(t, applyRepo.applied["comercio"], "apply must not reach the repository on invalid preset")
}

func TestApplyRejectsMissingRequiredModule(t *testing.T) {
	payload := testPlaybookPayload(t, nil)
	pb := testPlaybook(t, payload, []string{"tickets"})
	svc, _, applyRepo := newTestService(pb, &fakeModuleSvc{orgMods: []*registryDomain.OrganizationModule{}})

	_, err := svc.Apply(context.Background(), 1, "comercio")
	require.ErrorIs(t, err, domain.ErrPlaybookRequiresModules)
	assert.Zero(t, applyRepo.applied["comercio"])
}

func TestApplySkipsPresetForUnknownModule(t *testing.T) {
	payload := testPlaybookPayload(t, map[string]map[string]any{
		"no-existe": {"x": 1},
	})
	pb := testPlaybook(t, payload, nil)
	svc, _, applyRepo := newTestService(pb, &fakeModuleSvc{orgMods: []*registryDomain.OrganizationModule{}})

	_, err := svc.Apply(context.Background(), 1, "comercio")
	require.ErrorIs(t, err, domain.ErrInvalidModuleConfigPreset)
	assert.Zero(t, applyRepo.applied["comercio"])
}

func TestResetRemovesSeededState(t *testing.T) {
	payload := testPlaybookPayload(t, nil)
	pb := testPlaybook(t, payload, nil)
	svc, orgRepo, applyRepo := newTestService(pb, &fakeModuleSvc{})
	orgRepo.states["comercio"] = &domain.OrganizationPlaybook{OrganizationID: 1, PlaybookKey: "comercio"}
	applyRepo.applied["comercio"] = 1

	err := svc.Reset(context.Background(), 1, "comercio")
	require.NoError(t, err)
	assert.Zero(t, applyRepo.applied["comercio"])
}

func TestResetUnknownPlaybookReturnsNotFound(t *testing.T) {
	payload := testPlaybookPayload(t, nil)
	pb := testPlaybook(t, payload, nil)
	svc, _, _ := newTestService(pb, &fakeModuleSvc{})

	err := svc.Reset(context.Background(), 1, "nope")
	require.ErrorIs(t, err, domain.ErrPlaybookNotFound)
}

func TestListCatalogIncludesAppliedGuiones(t *testing.T) {
	payload := testPlaybookPayload(t, nil)
	pb := testPlaybook(t, payload, nil)
	svc, orgRepo, _ := newTestService(pb, &fakeModuleSvc{})
	orgRepo.states["comercio"] = &domain.OrganizationPlaybook{OrganizationID: 1, PlaybookKey: "comercio"}

	entries, err := svc.ListCatalog(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.True(t, entries[0].Applied != nil)
	assert.Len(t, entries[0].Guiones, 1)
	assert.Equal(t, "Saludo", entries[0].Guiones[0].Titulo)
}
