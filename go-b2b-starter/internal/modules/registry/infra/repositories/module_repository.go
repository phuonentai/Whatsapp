package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/registry/domain"
)

type moduleRepository struct {
	store sqlc.Store
}

func NewModuleRepository(store sqlc.Store) domain.ModuleRepository {
	return &moduleRepository{store: store}
}

func (r *moduleRepository) ListActive(ctx context.Context) ([]*domain.Module, error) {
	rows, err := r.store.ListActiveModules(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active modules: %w", err)
	}
	modules := make([]*domain.Module, len(rows))
	for i := range rows {
		modules[i] = mapModule(&rows[i])
	}
	return modules, nil
}

func (r *moduleRepository) GetByKey(ctx context.Context, key string) (*domain.Module, error) {
	row, err := r.store.GetModuleByKey(ctx, key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrModuleNotFound
		}
		return nil, fmt.Errorf("get module by key: %w", err)
	}
	return mapModule(&row), nil
}

func mapModule(m *sqlc.ModulesModule) *domain.Module {
	var granted, requires []string
	_ = json.Unmarshal(m.GrantedFeatures, &granted)
	_ = json.Unmarshal(m.Requires, &requires)
	return &domain.Module{
		ID:              m.ID,
		Key:             m.Key,
		Name:            m.Name,
		Description:     helpers.FromPgText(m.Description),
		GrantedFeatures: granted,
		Requires:        requires,
		ConfigSchema:    m.ConfigSchema,
		IsInternal:      m.IsInternal,
		IsActive:        m.IsActive,
	}
}
