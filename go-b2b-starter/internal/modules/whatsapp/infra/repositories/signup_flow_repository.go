package repositories

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
)

type signupFlowRepository struct {
	store sqlc.Store
}

func NewSignupFlowRepository(store sqlc.Store) domain.SignupFlowRepository {
	return &signupFlowRepository{store: store}
}

func (r *signupFlowRepository) Upsert(ctx context.Context, flow *domain.SignupFlow) (*domain.SignupFlow, error) {
	params := sqlc.UpsertWhatsAppSignupFlowParams{
		OrganizationID: flow.OrganizationID,
		Status:         string(flow.Status),
		Step:           helpers.ToPgText(flow.Step),
		ErrorCode:      helpers.ToPgText(flow.ErrorCode),
		RetryCount:     int32(flow.RetryCount),
		Metadata:       helpers.ToJSONB(flow.Metadata),
	}

	result, err := r.store.UpsertWhatsAppSignupFlow(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert signup flow: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *signupFlowRepository) GetByOrganization(ctx context.Context, orgID int32) (*domain.SignupFlow, error) {
	result, err := r.store.GetWhatsAppSignupFlowByOrganization(ctx, orgID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("%w: %w", domain.ErrSignupNotFound, err)
		}
		return nil, fmt.Errorf("failed to get signup flow: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *signupFlowRepository) UpdateStatus(ctx context.Context, orgID int32, status domain.SignupStatus, step, errorCode string, retryCount int, metadata map[string]interface{}) (*domain.SignupFlow, error) {
	params := sqlc.UpdateWhatsAppSignupFlowStatusParams{
		OrganizationID: orgID,
		Status:         string(status),
		Step:           helpers.ToPgText(step),
		ErrorCode:      helpers.ToPgText(errorCode),
		RetryCount:     int32(retryCount),
		Metadata:       helpers.ToJSONB(metadata),
	}

	result, err := r.store.UpdateWhatsAppSignupFlowStatus(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update signup flow status: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *signupFlowRepository) mapToDomain(f *sqlc.WhatsappSignupFlow) *domain.SignupFlow {
	return &domain.SignupFlow{
		ID:             f.ID,
		OrganizationID: f.OrganizationID,
		Status:         domain.SignupStatus(f.Status),
		Step:           helpers.FromPgText(f.Step),
		ErrorCode:      helpers.FromPgText(f.ErrorCode),
		RetryCount:     int(f.RetryCount),
		Metadata:       helpers.FromJSONB(f.Metadata),
		CreatedAt:      f.CreatedAt.Time,
		UpdatedAt:      f.UpdatedAt.Time,
	}
}
