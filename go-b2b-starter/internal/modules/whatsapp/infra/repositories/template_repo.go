package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
)

type templateRepository struct {
	store sqlc.Store
}

func NewTemplateRepository(store sqlc.Store) domain.TemplateRepository {
	return &templateRepository{store: store}
}

func (r *templateRepository) Create(ctx context.Context, t *domain.Template) (*domain.Template, error) {
	row, err := r.store.InsertWhatsAppTemplate(ctx, sqlc.InsertWhatsAppTemplateParams{
		OrganizationID: t.OrganizationID,
		Name:           t.Name,
		Category:       t.Category,
		Language:       t.Language,
		Body:           t.Body,
		ParamCount:     int32(t.ParamCount),
		Status:         string(t.Status),
		MetaTemplateID: helpers.ToPgTextPtr(t.MetaTemplateID),
		RejectionReason: helpers.ToPgTextPtr(t.RejectionReason),
		IsActive:       t.IsActive,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrTemplateNameConflict
		}
		return nil, fmt.Errorf("failed to insert template: %w", err)
	}
	return mapTemplate(&row), nil
}

func (r *templateRepository) GetByID(ctx context.Context, orgID int32, id int64) (*domain.Template, error) {
	row, err := r.store.GetWhatsAppTemplateByID(ctx, sqlc.GetWhatsAppTemplateByIDParams{
		ID:             id,
		OrganizationID: orgID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTemplateNotFound
		}
		return nil, fmt.Errorf("failed to get template by id: %w", err)
	}
	return mapTemplate(&row), nil
}

func (r *templateRepository) GetByOrgNameLanguage(ctx context.Context, orgID int32, name, language string) (*domain.Template, error) {
	row, err := r.store.GetWhatsAppTemplateByOrgAndNameLanguage(ctx, sqlc.GetWhatsAppTemplateByOrgAndNameLanguageParams{
		OrganizationID: orgID,
		Name:           name,
		Language:       language,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTemplateNotFound
		}
		return nil, fmt.Errorf("failed to get template by name/language: %w", err)
	}
	return mapTemplate(&row), nil
}

func (r *templateRepository) GetByMetaTemplateID(ctx context.Context, orgID int32, metaTemplateID string) (*domain.Template, error) {
	row, err := r.store.GetWhatsAppTemplateByMetaTemplateID(ctx, sqlc.GetWhatsAppTemplateByMetaTemplateIDParams{
		MetaTemplateID: helpers.ToPgTextPtr(&metaTemplateID),
		OrganizationID: orgID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTemplateNotFound
		}
		return nil, fmt.Errorf("failed to get template by meta id: %w", err)
	}
	return mapTemplate(&row), nil
}

func (r *templateRepository) ListByOrg(ctx context.Context, orgID int32) ([]*domain.Template, error) {
	rows, err := r.store.ListWhatsAppTemplatesByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}
	out := make([]*domain.Template, 0, len(rows))
	for i := range rows {
		out = append(out, mapTemplate(&rows[i]))
	}
	return out, nil
}

func (r *templateRepository) Update(ctx context.Context, t *domain.Template) (*domain.Template, error) {
	row, err := r.store.UpdateWhatsAppTemplate(ctx, sqlc.UpdateWhatsAppTemplateParams{
		ID:             t.ID,
		OrganizationID: t.OrganizationID,
		Name:           t.Name,
		Category:       t.Category,
		Language:       t.Language,
		Body:           t.Body,
		ParamCount:     int32(t.ParamCount),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTemplateNotFound
		}
		if isUniqueViolation(err) {
			return nil, domain.ErrTemplateNameConflict
		}
		return nil, fmt.Errorf("failed to update template: %w", err)
	}
	return mapTemplate(&row), nil
}

func (r *templateRepository) UpdateStatus(ctx context.Context, orgID int32, id int64, status domain.TemplateStatus, metaTemplateID, rejectionReason *string) (*domain.Template, error) {
	row, err := r.store.UpdateWhatsAppTemplateStatus(ctx, sqlc.UpdateWhatsAppTemplateStatusParams{
		ID:              id,
		OrganizationID:  orgID,
		Status:          string(status),
		MetaTemplateID:  helpers.ToPgTextPtr(metaTemplateID),
		RejectionReason: helpers.ToPgTextPtr(rejectionReason),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No row returned: either the template does not exist for the org
			// or the stored status already equals the target (idempotent
			// no-op). The caller distinguishes via a prior existence check.
			return nil, nil
		}
		return nil, fmt.Errorf("failed to update template status: %w", err)
	}
	return mapTemplate(&row), nil
}

func (r *templateRepository) Delete(ctx context.Context, orgID int32, id int64) error {
	if _, err := r.store.DeleteWhatsAppTemplate(ctx, sqlc.DeleteWhatsAppTemplateParams{
		ID:             id,
		OrganizationID: orgID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrTemplateNotFound
		}
		return fmt.Errorf("failed to delete template: %w", err)
	}
	return nil
}

func (r *templateRepository) CountByOrg(ctx context.Context, orgID int32) (int64, error) {
	count, err := r.store.CountWhatsAppTemplatesByOrg(ctx, orgID)
	if err != nil {
		return 0, fmt.Errorf("failed to count templates: %w", err)
	}
	return count, nil
}

func mapTemplate(row *sqlc.WhatsappTemplate) *domain.Template {
	return &domain.Template{
		ID:              row.ID,
		OrganizationID:  row.OrganizationID,
		Name:            row.Name,
		Category:        row.Category,
		Language:        row.Language,
		Body:            row.Body,
		ParamCount:      int(row.ParamCount),
		Status:          domain.TemplateStatus(row.Status),
		MetaTemplateID:  helpers.FromPgTextPtr(row.MetaTemplateID),
		RejectionReason: helpers.FromPgTextPtr(row.RejectionReason),
		IsActive:        row.IsActive,
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
