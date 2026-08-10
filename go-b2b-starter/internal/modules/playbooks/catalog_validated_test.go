package playbooks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/moasq/go-b2b-starter/internal/modules/playbooks/domain"
)

// fakeCatalogRepo returns playbooks with payloads equal to the Go catalog
// (the startup check treats those rows as "the DB seed").
type fakeCatalogRepo struct {
	rows []*domain.Playbook
}

func (f *fakeCatalogRepo) ListActive(ctx context.Context) ([]*domain.Playbook, error) {
	return f.rows, nil
}

func (f *fakeCatalogRepo) GetByKey(ctx context.Context, key string) (*domain.Playbook, error) {
	for _, pb := range f.rows {
		if pb.Key == key {
			return pb, nil
		}
	}
	return nil, domain.ErrPlaybookNotFound
}

func (f *fakeCatalogRepo) ListByOrg(ctx context.Context, orgID int32) ([]*domain.OrganizationPlaybook, error) {
	return nil, nil
}

func (f *fakeCatalogRepo) GetByOrgKey(ctx context.Context, orgID int32, key string) (*domain.OrganizationPlaybook, error) {
	return nil, nil
}

func (f *fakeCatalogRepo) Upsert(ctx context.Context, orgID int32, key string, seededPipelineID *int32) (*domain.OrganizationPlaybook, error) {
	return nil, nil
}

func (f *fakeCatalogRepo) Delete(ctx context.Context, orgID int32, key string) error {
	return nil
}

func (f *fakeCatalogRepo) Apply(ctx context.Context, orgID int32, pb *domain.Playbook, payload *domain.PlaybookPayload) (*domain.OrganizationPlaybook, error) {
	return nil, nil
}

func (f *fakeCatalogRepo) Reset(ctx context.Context, orgID int32, pb *domain.Playbook, payload *domain.PlaybookPayload) error {
	return nil
}

func seededFromCatalog() []*domain.Playbook {
	catalog := Catalog()
	rows := make([]*domain.Playbook, 0, len(catalog))
	for i := range catalog {
		cp := catalog[i]
		rows = append(rows, &cp)
	}
	return rows
}

func TestCatalogValidatedMatchesSeededCatalog(t *testing.T) {
	repo := &fakeCatalogRepo{rows: seededFromCatalog()}
	require.NoError(t, CatalogValidated(context.Background(), repo))
}

func TestCatalogValidatedFailsOnMissingVertical(t *testing.T) {
	rows := seededFromCatalog()
	rows = rows[:len(rows)-1]
	repo := &fakeCatalogRepo{rows: rows}

	err := CatalogValidated(context.Background(), repo)
	require.ErrorIs(t, err, ErrCatalogValidationFailed)
	require.ErrorContains(t, err, "missing from DB seed")
}

func TestCatalogValidatedFailsOnStaleSingleShotGuion(t *testing.T) {
	rows := seededFromCatalog()
	for i, pb := range rows {
		if pb.Key != "comercio" {
			continue
		}
		rows[i] = &domain.Playbook{
			Key:      pb.Key,
			Name:     pb.Name,
			Vertical: pb.Vertical,
			Payload: payloadJSON(map[string]any{
				"pipeline": map[string]any{
					"nombre": "Ventas por WhatsApp",
					"etapas": []map[string]any{
						{"nombre": "Nuevo Pedido", "orden": 1, "color": "#6B7280", "probabilidad": 10},
					},
				},
				"tags":           []string{"nuevo"},
				"module_configs": map[string]any{},
				"guiones": []map[string]any{
					{"id": "confirmar-pedido", "titulo": "Confirmar pedido", "mensaje": "¡Claro! Confirmamos tu pedido. ¿Quieres que te enviemos el link de pago?"},
				},
			}),
		}
	}
	repo := &fakeCatalogRepo{rows: rows}

	err := CatalogValidated(context.Background(), repo)
	require.ErrorIs(t, err, ErrCatalogValidationFailed)
	require.ErrorContains(t, err, "guiones drift")
}

func TestCatalogValidatedFailsOnUnknownSeededKey(t *testing.T) {
	rows := seededFromCatalog()
	rows = append(rows, &domain.Playbook{Key: "misterio", Vertical: "x", Name: "Misterio", Payload: []byte(`{}`)})
	repo := &fakeCatalogRepo{rows: rows}

	err := CatalogValidated(context.Background(), repo)
	require.ErrorIs(t, err, ErrCatalogValidationFailed)
	require.ErrorContains(t, err, "missing from catalog")
}
