package services

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moasq/go-b2b-starter/internal/modules/registry/domain"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

type fakeModuleRepo struct {
	modules []*domain.Module
}

func (f *fakeModuleRepo) ListActive(ctx context.Context) ([]*domain.Module, error) { return f.modules, nil }
func (f *fakeModuleRepo) GetByKey(ctx context.Context, key string) (*domain.Module, error) {
	for _, m := range f.modules {
		if m.Key == key {
			return m, nil
		}
	}
	return nil, domain.ErrModuleNotFound
}

type fakeOrgModRepo struct {
	orgMods []*domain.OrganizationModule
	deleted []string
}

func (f *fakeOrgModRepo) ListByOrg(ctx context.Context, orgID int32) ([]*domain.OrganizationModule, error) {
	return f.orgMods, nil
}
func (f *fakeOrgModRepo) GetByKey(ctx context.Context, orgID int32, moduleKey string) (*domain.OrganizationModule, error) {
	for _, om := range f.orgMods {
		if om.ModuleKey == moduleKey {
			return om, nil
		}
	}
	return nil, nil
}
func (f *fakeOrgModRepo) UpsertConfig(ctx context.Context, orgID int32, moduleKey string, config map[string]any) (*domain.OrganizationModule, error) {
	return &domain.OrganizationModule{OrganizationID: orgID, ModuleKey: moduleKey, Config: config}, nil
}
func (f *fakeOrgModRepo) Delete(ctx context.Context, orgID int32, moduleKey string) error {
	f.deleted = append(f.deleted, moduleKey)
	return nil
}

type fakeLogger struct{}

func (f fakeLogger) Debug(msg string, fields ...map[string]any) {}
func (f fakeLogger) Info(msg string, fields ...map[string]any)  {}
func (f fakeLogger) Warn(msg string, fields ...map[string]any)  {}
func (f fakeLogger) Error(msg string, fields ...map[string]any) {}
func (f fakeLogger) Fatal(msg string, fields ...map[string]any) {}
func (f fakeLogger) WithFields(fields map[string]any) loggerDomain.Logger { return f }

func newFakeModuleSet() []*domain.Module {
	return []*domain.Module{
		{
			Key: "tickets", Name: "Tickets", GrantedFeatures: []string{"tickets_module"},
			Requires: []string{}, IsActive: true, ConfigSchema: json.RawMessage(`{}`),
		},
		{
			Key: "ops-internal", Name: "Ops Interno", GrantedFeatures: []string{"ops_tools"},
			Requires: []string{}, IsActive: true, IsInternal: true, ConfigSchema: json.RawMessage(`{}`),
		},
		{
			Key: "analytics", Name: "Analytics", GrantedFeatures: []string{"analytics_module"},
			Requires: []string{"tickets"}, IsActive: true, ConfigSchema: json.RawMessage(`{}`),
		},
	}
}

func TestListCatalog_ExcludesInternalModules(t *testing.T) {
	svc := NewModuleService(&fakeModuleRepo{modules: newFakeModuleSet()}, &fakeOrgModRepo{}, fakeLogger{})
	catalog, err := svc.ListCatalog(context.Background())
	require.NoError(t, err)
	for _, m := range catalog {
		assert.False(t, m.IsInternal, "internal modules must not be tenant-visible")
	}
	assert.Len(t, catalog, 2)
}

func TestResolveGrantedFeatures_RespectsDependencies(t *testing.T) {
	svc := NewModuleService(&fakeModuleRepo{modules: newFakeModuleSet()}, &fakeOrgModRepo{}, fakeLogger{})

	// analytics requires tickets: without tickets, analytics features must not be granted.
	features := svc.ResolveGrantedFeatures(context.Background(), []string{"analytics"})
	assert.False(t, features["analytics_module"])

	// With both enabled, both module features are granted.
	features = svc.ResolveGrantedFeatures(context.Background(), []string{"tickets", "analytics"})
	assert.True(t, features["tickets_module"])
	assert.True(t, features["analytics_module"])

	// Unknown module keys are ignored.
	features = svc.ResolveGrantedFeatures(context.Background(), []string{"ghost"})
	assert.Len(t, features, 0)
}

func TestSaveOrgModuleConfig_ValidatesSchema(t *testing.T) {
	svc := NewModuleService(&fakeModuleRepo{modules: newFakeModuleSet()}, &fakeOrgModRepo{}, fakeLogger{})

	_, err := svc.SaveOrgModuleConfig(context.Background(), 1, "tickets", map[string]any{
		"sla_hours": "not-an-object",
	})
	require.ErrorIs(t, err, domain.ErrInvalidModuleConfig)

	_, err = svc.SaveOrgModuleConfig(context.Background(), 1, "tickets", map[string]any{
		"sla_hours":   map[string]any{"high": float64(2)},
		"priorities":  []any{"low", "normal"},
		"tags":        []any{"billing"},
	})
	require.NoError(t, err)

	_, err = svc.SaveOrgModuleConfig(context.Background(), 1, "ghost", map[string]any{})
	require.ErrorIs(t, err, domain.ErrModuleNotFound)
}

func TestSyncModulesFromMetadata_EmptyKeysAreNoOp(t *testing.T) {
	modRepo := &fakeModuleRepo{modules: newFakeModuleSet()}
	orgModRepo := &fakeOrgModRepo{orgMods: []*domain.OrganizationModule{
		{OrganizationID: 7, ModuleKey: "tickets", Config: map[string]any{"sla_hours": float64(4)}},
	}}
	svc := NewModuleService(modRepo, orgModRepo, fakeLogger{})

	// Absent/empty metadata key sets must not disable existing org modules.
	err := svc.SyncModulesFromMetadata(context.Background(), 7, nil)
	require.NoError(t, err)
	err = svc.SyncModulesFromMetadata(context.Background(), 7, []string{})
	require.NoError(t, err)

	assert.Empty(t, orgModRepo.deleted, "no module may be disabled on empty metadata")
	require.Len(t, orgModRepo.orgMods, 1)
	assert.Equal(t, "tickets", orgModRepo.orgMods[0].ModuleKey)
	assert.Equal(t, map[string]any{"sla_hours": float64(4)}, orgModRepo.orgMods[0].Config)
}
