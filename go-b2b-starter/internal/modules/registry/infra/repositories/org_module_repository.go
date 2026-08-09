package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/registry/domain"
)

type orgModuleRepository struct {
	store sqlc.Store
}

func NewOrgModuleRepository(store sqlc.Store) domain.OrganizationModuleRepository {
	return &orgModuleRepository{store: store}
}

func (r *orgModuleRepository) ListByOrg(ctx context.Context, orgID int32) ([]*domain.OrganizationModule, error) {
	rows, err := r.store.ListOrgModules(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list org modules: %w", err)
	}
	result := make([]*domain.OrganizationModule, len(rows))
	for i := range rows {
		result[i] = mapOrgModule(&rows[i])
	}
	return result, nil
}

func (r *orgModuleRepository) GetByKey(ctx context.Context, orgID int32, moduleKey string) (*domain.OrganizationModule, error) {
	row, err := r.store.GetOrgModule(ctx, sqlc.GetOrgModuleParams{
		OrganizationID: orgID,
		ModuleKey:      moduleKey,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get org module: %w", err)
	}
	return mapOrgModule(&row), nil
}

func (r *orgModuleRepository) UpsertConfig(ctx context.Context, orgID int32, moduleKey string, config map[string]any) (*domain.OrganizationModule, error) {
	row, err := r.store.UpsertOrgModule(ctx, sqlc.UpsertOrgModuleParams{
		OrganizationID: orgID,
		ModuleKey:      moduleKey,
		Config:         helpers.ToJSONB(config),
	})
	if err != nil {
		return nil, fmt.Errorf("upsert org module: %w", err)
	}
	return mapOrgModule(&row), nil
}

func (r *orgModuleRepository) Delete(ctx context.Context, orgID int32, moduleKey string) error {
	if err := r.store.DeleteOrgModule(ctx, sqlc.DeleteOrgModuleParams{
		OrganizationID: orgID,
		ModuleKey:      moduleKey,
	}); err != nil {
		return fmt.Errorf("delete org module: %w", err)
	}
	return nil
}

func mapOrgModule(om *sqlc.ModulesOrganizationModule) *domain.OrganizationModule {
	return &domain.OrganizationModule{
		OrganizationID: om.OrganizationID,
		ModuleKey:      om.ModuleKey,
		Config:         helpers.FromJSONB(om.Config),
		EnabledAt:      om.EnabledAt.Time.String(),
	}
}
