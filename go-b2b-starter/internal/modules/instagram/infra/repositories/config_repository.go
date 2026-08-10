package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/instagram/domain"
)

type configRepository struct {
	store sqlc.Store
}

func NewConfigRepository(store sqlc.Store) domain.ConfigRepository {
	return &configRepository{store: store}
}

func (r *configRepository) GetByIGUserID(ctx context.Context, igUserID string) (*domain.InstagramConfig, error) {
	result, err := r.store.GetInstagramConfigByIGUserID(ctx, igUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrConfigNotFound
		}
		return nil, fmt.Errorf("failed to get config by IG user ID: %w", err)
	}
	return r.mapToDomain(&result), nil
}

func (r *configRepository) GetByOrganizationID(ctx context.Context, orgID int32) (*domain.InstagramConfig, error) {
	result, err := r.store.GetInstagramConfigByOrganizationID(ctx, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrConfigNotFound
		}
		return nil, fmt.Errorf("failed to get config by organization ID: %w", err)
	}
	return r.mapToDomain(&result), nil
}

func (r *configRepository) GetByVerifyToken(ctx context.Context, verifyToken string) (*domain.InstagramConfig, error) {
	result, err := r.store.GetInstagramConfigByVerifyToken(ctx, verifyToken)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrConfigNotFound
		}
		return nil, fmt.Errorf("failed to get config by verify token: %w", err)
	}
	return r.mapToDomain(&result), nil
}

func (r *configRepository) Create(ctx context.Context, config *domain.InstagramConfig) (*domain.InstagramConfig, error) {
	params := sqlc.CreateInstagramConfigParams{
		OrganizationID: config.OrganizationID,
		IgUserID:       config.IGUserID,
		IgUsername:     helpers.ToPgText(config.IGUsername),
		FbPageID:       helpers.ToPgText(config.FBPageID),
		AccessToken:    config.AccessToken,
		TokenExpiresAt: helpers.ToPgTimestamptzPtr(config.TokenExpiresAt),
		WebhookSecret:  config.WebhookSecret,
		VerifyToken:    config.VerifyToken,
		ApiVersion:     config.APIVersion,
		GraphApiUrl:    config.GraphAPIURL,
		Metadata:       helpers.ToJSONB(config.Metadata),
	}

	result, err := r.store.CreateInstagramConfig(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create instagram config: %w", err)
	}
	return r.mapToDomain(&result), nil
}

func (r *configRepository) Update(ctx context.Context, config *domain.InstagramConfig) (*domain.InstagramConfig, error) {
	params := sqlc.UpdateInstagramConfigParams{
		ID:             config.ID,
		IgUserID:       config.IGUserID,
		IgUsername:     helpers.ToPgText(config.IGUsername),
		FbPageID:       helpers.ToPgText(config.FBPageID),
		AccessToken:    config.AccessToken,
		TokenExpiresAt: helpers.ToPgTimestamptzPtr(config.TokenExpiresAt),
		WebhookSecret:  config.WebhookSecret,
		VerifyToken:    config.VerifyToken,
		ApiVersion:     config.APIVersion,
		GraphApiUrl:    config.GraphAPIURL,
		IsActive:       config.IsActive,
		Metadata:       helpers.ToJSONB(config.Metadata),
	}

	result, err := r.store.UpdateInstagramConfig(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update instagram config: %w", err)
	}
	return r.mapToDomain(&result), nil
}

func (r *configRepository) mapToDomain(c *sqlc.WhatsappInstagramConfig) *domain.InstagramConfig {
	return &domain.InstagramConfig{
		ID:             c.ID,
		OrganizationID: c.OrganizationID,
		IGUserID:       c.IgUserID,
		IGUsername:     helpers.FromPgText(c.IgUsername),
		FBPageID:       helpers.FromPgText(c.FbPageID),
		AccessToken:    c.AccessToken,
		TokenExpiresAt: helpers.FromPgTimestamptzPtr(c.TokenExpiresAt),
		WebhookSecret:  c.WebhookSecret,
		VerifyToken:    c.VerifyToken,
		APIVersion:     c.ApiVersion,
		GraphAPIURL:    c.GraphApiUrl,
		IsActive:       c.IsActive,
		Metadata:       helpers.FromJSONB(c.Metadata),
		CreatedAt:      c.CreatedAt.Time,
		UpdatedAt:      c.UpdatedAt.Time,
	}
}
