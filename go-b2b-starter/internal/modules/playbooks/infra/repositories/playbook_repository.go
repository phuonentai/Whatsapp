package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/playbooks/domain"
)

// transactioner is satisfied by *sqlc.SQLStore; unit tests use stores that
// implement it or fall back to direct execution.
type transactioner interface {
	Transaction(ctx context.Context, fn func(sqlc.Store) error) error
}

type PlaybookRepository struct {
	store sqlc.Store
}

func NewPlaybookRepository(store sqlc.Store) *PlaybookRepository {
	return &PlaybookRepository{store: store}
}

func NewOrgPlaybookRepository(store sqlc.Store) *PlaybookRepository {
	return &PlaybookRepository{store: store}
}

func (r *PlaybookRepository) inTx(ctx context.Context, fn func(sqlc.Store) error) error {
	if t, ok := r.store.(transactioner); ok {
		return t.Transaction(ctx, fn)
	}
	return fn(r.store)
}

func (r *PlaybookRepository) ListActive(ctx context.Context) ([]*domain.Playbook, error) {
	rows, err := r.store.ListActivePlaybooks(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active playbooks: %w", err)
	}
	result := make([]*domain.Playbook, len(rows))
	for i := range rows {
		result[i] = mapPlaybook(&rows[i])
	}
	return result, nil
}

func (r *PlaybookRepository) GetByKey(ctx context.Context, key string) (*domain.Playbook, error) {
	row, err := r.store.GetPlaybookByKey(ctx, key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPlaybookNotFound
		}
		return nil, fmt.Errorf("get playbook %q: %w", key, err)
	}
	return mapPlaybook(&row), nil
}

func (r *PlaybookRepository) ListByOrg(ctx context.Context, orgID int32) ([]*domain.OrganizationPlaybook, error) {
	rows, err := r.store.ListOrgPlaybooks(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list org playbooks: %w", err)
	}
	result := make([]*domain.OrganizationPlaybook, len(rows))
	for i := range rows {
		result[i] = mapOrgPlaybook(&rows[i])
	}
	return result, nil
}

func (r *PlaybookRepository) GetByOrgKey(ctx context.Context, orgID int32, key string) (*domain.OrganizationPlaybook, error) {
	row, err := r.store.GetOrgPlaybook(ctx, sqlc.GetOrgPlaybookParams{
		OrganizationID: orgID,
		PlaybookKey:    key,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get org playbook: %w", err)
	}
	return mapOrgPlaybook(&row), nil
}

func (r *PlaybookRepository) Upsert(ctx context.Context, orgID int32, key string, seededPipelineID *int32) (*domain.OrganizationPlaybook, error) {
	row, err := r.store.UpsertOrgPlaybook(ctx, sqlc.UpsertOrgPlaybookParams{
		OrganizationID:   orgID,
		PlaybookKey:      key,
		SeededPipelineID: helpers.ToPgInt4Ptr(seededPipelineID),
	})
	if err != nil {
		return nil, fmt.Errorf("upsert org playbook: %w", err)
	}
	return mapOrgPlaybook(&row), nil
}

func (r *PlaybookRepository) Delete(ctx context.Context, orgID int32, key string) error {
	if err := r.store.DeleteOrgPlaybook(ctx, sqlc.DeleteOrgPlaybookParams{
		OrganizationID: orgID,
		PlaybookKey:    key,
	}); err != nil {
		return fmt.Errorf("delete org playbook: %w", err)
	}
	return nil
}

// Apply seeds the playbook's pipeline, tags, and module config presets in one
// transaction. One-way and additive: nothing is deleted or overwritten; config
// presets merge only missing keys into the org's stored config.
func (r *PlaybookRepository) Apply(ctx context.Context, orgID int32, pb *domain.Playbook, payload *domain.PlaybookPayload) (*domain.OrganizationPlaybook, error) {
	var result *domain.OrganizationPlaybook
	err := r.inTx(ctx, func(tx sqlc.Store) error {
		var seededPipelineID *int32

		pipelines, err := tx.ListPipelinesByOrganization(ctx, orgID)
		if err != nil {
			return fmt.Errorf("list org pipelines: %w", err)
		}
		if len(pipelines) == 0 {
			pipeline, err := tx.CreatePipeline(ctx, sqlc.CreatePipelineParams{
				OrganizationID:   orgID,
				Nombre:           payload.Pipeline.Nombre,
				EsPredeterminado: true,
				Orden:            1,
			})
			if err != nil {
				return fmt.Errorf("create playbook pipeline: %w", err)
			}
			for _, etapa := range payload.Pipeline.Etapas {
				if _, err := tx.CreatePipelineStage(ctx, sqlc.CreatePipelineStageParams{
					PipelineID:   pipeline.ID,
					Nombre:       etapa.Nombre,
					Orden:        etapa.Orden,
					Color:        helpers.ToPgText(etapa.Color),
					Probabilidad: helpers.ToPgInt4(etapa.Probabilidad),
				}); err != nil {
					return fmt.Errorf("create playbook pipeline stage %q: %w", etapa.Nombre, err)
				}
			}
			seededPipelineID = &pipeline.ID
		}

		if err := r.seedTags(ctx, tx, orgID, payload.Tags); err != nil {
			return err
		}

		if err := r.seedModuleConfigs(ctx, tx, orgID, payload.ModuleConfigs); err != nil {
			return err
		}

		state, err := tx.UpsertOrgPlaybook(ctx, sqlc.UpsertOrgPlaybookParams{
			OrganizationID:   orgID,
			PlaybookKey:      pb.Key,
			SeededPipelineID: helpers.ToPgInt4Ptr(seededPipelineID),
		})
		if err != nil {
			return fmt.Errorf("upsert org playbook: %w", err)
		}
		result = mapOrgPlaybook(&state)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Reset removes playbook-seeded data only: config presets still matching the
// preset, unreferenced seeded tags, and the seeded pipeline when it has no
// deals. Organization-created data is never touched.
func (r *PlaybookRepository) Reset(ctx context.Context, orgID int32, pb *domain.Playbook, payload *domain.PlaybookPayload) error {
	return r.inTx(ctx, func(tx sqlc.Store) error {
		state, err := tx.GetOrgPlaybook(ctx, sqlc.GetOrgPlaybookParams{
			OrganizationID: orgID,
			PlaybookKey:    pb.Key,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return fmt.Errorf("get org playbook: %w", err)
		}

		if err := r.revertModuleConfigs(ctx, tx, orgID, payload.ModuleConfigs); err != nil {
			return err
		}
		if err := r.removeTags(ctx, tx, orgID, payload.Tags); err != nil {
			return err
		}

		if state.SeededPipelineID.Valid {
			deals, err := tx.ListDealsByOrganization(ctx, sqlc.ListDealsByOrganizationParams{
				OrganizationID: orgID,
				Column2:        state.SeededPipelineID.Int32,
				Limit:          1,
			})
			if err != nil {
				return fmt.Errorf("list deals for seeded pipeline: %w", err)
			}
			if len(deals) == 0 {
				if err := tx.DeletePipeline(ctx, sqlc.DeletePipelineParams{
					ID:             state.SeededPipelineID.Int32,
					OrganizationID: orgID,
				}); err != nil {
					return fmt.Errorf("delete seeded pipeline: %w", err)
				}
			}
		}

		if err := tx.DeleteOrgPlaybook(ctx, sqlc.DeleteOrgPlaybookParams{
			OrganizationID: orgID,
			PlaybookKey:    pb.Key,
		}); err != nil {
			return fmt.Errorf("delete org playbook state: %w", err)
		}
		return nil
	})
}

func (r *PlaybookRepository) seedTags(ctx context.Context, tx sqlc.Store, orgID int32, tags []string) error {
	existing, err := tx.ListTagsByOrganization(ctx, orgID)
	if err != nil {
		return fmt.Errorf("list org tags: %w", err)
	}
	have := make(map[string]bool, len(existing))
	for _, t := range existing {
		have[t.Nombre] = true
	}
	for _, name := range tags {
		if have[name] {
			continue
		}
		if _, err := tx.CreateTag(ctx, sqlc.CreateTagParams{
			OrganizationID: orgID,
			Nombre:         name,
			Color:          helpers.ToPgText(""),
		}); err != nil {
			return fmt.Errorf("create playbook tag %q: %w", name, err)
		}
	}
	return nil
}

func (r *PlaybookRepository) seedModuleConfigs(ctx context.Context, tx sqlc.Store, orgID int32, presets map[string]map[string]any) error {
	orgMods, err := tx.ListOrgModules(ctx, orgID)
	if err != nil {
		return fmt.Errorf("list org modules: %w", err)
	}
	enabled := make(map[string]bool, len(orgMods))
	for _, om := range orgMods {
		enabled[om.ModuleKey] = true
	}
	for moduleKey, preset := range presets {
		if !enabled[moduleKey] {
			continue
		}
		existingRow, err := tx.GetOrgModule(ctx, sqlc.GetOrgModuleParams{
			OrganizationID: orgID,
			ModuleKey:      moduleKey,
		})
		var existing *sqlc.ModulesOrganizationModule
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				existing = nil
			} else {
				return fmt.Errorf("get org module %q: %w", moduleKey, err)
			}
		} else {
			existing = &existingRow
		}
		merged := map[string]any{}
		if existing != nil {
			merged = helpers.FromJSONB(existing.Config)
		}
		for k, v := range preset {
			if _, ok := merged[k]; !ok {
				merged[k] = v
			}
		}
		if _, err := tx.UpsertOrgModule(ctx, sqlc.UpsertOrgModuleParams{
			OrganizationID: orgID,
			ModuleKey:      moduleKey,
			Config:         helpers.ToJSONB(merged),
		}); err != nil {
			return fmt.Errorf("apply module config preset for %q: %w", moduleKey, err)
		}
	}
	return nil
}

func (r *PlaybookRepository) revertModuleConfigs(ctx context.Context, tx sqlc.Store, orgID int32, presets map[string]map[string]any) error {
	for moduleKey, preset := range presets {
		existingRow, err := tx.GetOrgModule(ctx, sqlc.GetOrgModuleParams{
			OrganizationID: orgID,
			ModuleKey:      moduleKey,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return fmt.Errorf("get org module %q: %w", moduleKey, err)
		}
		if !jsonEqual(existingRow.Config, preset) {
			continue
		}
		if _, err := tx.UpsertOrgModule(ctx, sqlc.UpsertOrgModuleParams{
			OrganizationID: orgID,
			ModuleKey:      moduleKey,
			Config:         helpers.ToJSONB(map[string]any{}),
		}); err != nil {
			return fmt.Errorf("revert module config preset for %q: %w", moduleKey, err)
		}
	}
	return nil
}

func (r *PlaybookRepository) removeTags(ctx context.Context, tx sqlc.Store, orgID int32, tags []string) error {
	existing, err := tx.ListTagsByOrganization(ctx, orgID)
	if err != nil {
		return fmt.Errorf("list org tags: %w", err)
	}
	byName := make(map[string]sqlc.CrmTag, len(existing))
	for _, t := range existing {
		byName[t.Nombre] = t
	}
	for _, name := range tags {
		tag, ok := byName[name]
		if !ok {
			continue
		}
		refs, err := tx.ListEntitiesByTag(ctx, tag.ID)
		if err != nil {
			return fmt.Errorf("list tag references %q: %w", name, err)
		}
		if len(refs) > 0 {
			continue
		}
		if err := tx.DeleteTag(ctx, sqlc.DeleteTagParams{
			ID:             tag.ID,
			OrganizationID: orgID,
		}); err != nil {
			return fmt.Errorf("delete seeded tag %q: %w", name, err)
		}
	}
	return nil
}

func mapPlaybook(row *sqlc.ModulesPlaybook) *domain.Playbook {
	return &domain.Playbook{
		ID:              row.ID,
		Key:             row.Key,
		Name:            row.Name,
		Vertical:        row.Vertical,
		Description:     row.Description.String,
		RequiresModules: helpers.FromJSONBStringSlice(row.RequiresModules),
		Payload:         row.Payload,
		IsActive:        row.IsActive,
	}
}

func mapOrgPlaybook(row *sqlc.ModulesOrganizationPlaybook) *domain.OrganizationPlaybook {
	appliedAt := row.AppliedAt.Time
	return &domain.OrganizationPlaybook{
		OrganizationID:   row.OrganizationID,
		PlaybookKey:      row.PlaybookKey,
		SeededPipelineID: helpers.FromPgInt4Ptr(row.SeededPipelineID),
		AppliedAt:        appliedAt.String(),
	}
}

// jsonEqual compares a stored JSONB config with a preset deterministically
// (encoding/json sorts map keys, so byte equality is stable).
func jsonEqual(raw []byte, preset map[string]any) bool {
	stored, err := json.Marshal(helpers.FromJSONB(raw))
	if err != nil {
		return false
	}
	expected, err := json.Marshal(preset)
	if err != nil {
		return false
	}
	return string(stored) == string(expected)
}
