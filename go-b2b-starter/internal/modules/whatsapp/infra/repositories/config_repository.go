package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
)

type configRepository struct {
	store sqlc.Store
}

func NewConfigRepository(store sqlc.Store) domain.ConfigRepository {
	return &configRepository{store: store}
}

func (r *configRepository) GetByPhoneNumberID(ctx context.Context, phoneNumberID string) (*domain.WhatsAppConfig, error) {
	result, err := r.store.GetWhatsAppConfigByPhoneNumberID(ctx, phoneNumberID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrConfigNotFound
		}
		return nil, fmt.Errorf("failed to get config by phone number ID: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *configRepository) GetByOrganizationID(ctx context.Context, orgID int32) (*domain.WhatsAppConfig, error) {
	result, err := r.store.GetWhatsAppConfigByOrganizationID(ctx, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrConfigNotFound
		}
		return nil, fmt.Errorf("failed to get config by organization ID: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *configRepository) GetByVerifyToken(ctx context.Context, verifyToken string) (*domain.WhatsAppConfig, error) {
	result, err := r.store.GetWhatsAppConfigByVerifyToken(ctx, verifyToken)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrConfigNotFound
		}
		return nil, fmt.Errorf("failed to get config by verify token: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *configRepository) GetByWABAID(ctx context.Context, wabaID string) (*domain.WhatsAppConfig, error) {
	result, err := r.store.GetWhatsAppConfigByWABAID(ctx, helpers.ToPgText(wabaID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrConfigNotFound
		}
		return nil, fmt.Errorf("failed to get config by waba id: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *configRepository) Create(ctx context.Context, config *domain.WhatsAppConfig) (*domain.WhatsAppConfig, error) {
	params := sqlc.CreateWhatsAppConfigParams{
		OrganizationID: config.OrganizationID,
		PhoneNumberID:  config.PhoneNumberID,
		BusinessPhone:  config.BusinessPhone,
		WebhookSecret:  config.WebhookSecret,
		VerifyToken:    config.VerifyToken,
		AppID:          helpers.ToPgText(config.AppID),
		WabaID:         helpers.ToPgText(config.WABAID),
		AccessToken:    helpers.ToPgText(config.AccessToken),
		ApiVersion:     config.APIVersion,
		GraphApiUrl:    config.GraphAPIURL,
		Metadata:       helpers.ToJSONB(config.Metadata),
	}

	result, err := r.store.CreateWhatsAppConfig(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *configRepository) Update(ctx context.Context, config *domain.WhatsAppConfig) (*domain.WhatsAppConfig, error) {
	params := sqlc.UpdateWhatsAppConfigParams{
		ID:            config.ID,
		PhoneNumberID: config.PhoneNumberID,
		BusinessPhone: config.BusinessPhone,
		WebhookSecret: config.WebhookSecret,
		VerifyToken:   config.VerifyToken,
		AppID:         helpers.ToPgText(config.AppID),
		WabaID:        helpers.ToPgText(config.WABAID),
		AccessToken:   helpers.ToPgText(config.AccessToken),
		ApiVersion:    config.APIVersion,
		GraphApiUrl:   config.GraphAPIURL,
		IsActive:      config.IsActive,
		Metadata:      helpers.ToJSONB(config.Metadata),
	}

	result, err := r.store.UpdateWhatsAppConfig(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	return r.mapToDomain(&result), nil
}

func (r *configRepository) mapToDomain(c *sqlc.WhatsappWhatsappConfig) *domain.WhatsAppConfig {
	return &domain.WhatsAppConfig{
		ID:             c.ID,
		OrganizationID: c.OrganizationID,
		PhoneNumberID:  c.PhoneNumberID,
		BusinessPhone:  c.BusinessPhone,
		WebhookSecret:  c.WebhookSecret,
		VerifyToken:    c.VerifyToken,
		AppID:          helpers.FromPgText(c.AppID),
		WABAID:         helpers.FromPgText(c.WabaID),
		AccessToken:    helpers.FromPgText(c.AccessToken),
		APIVersion:     c.ApiVersion,
		GraphAPIURL:    c.GraphApiUrl,
		IsActive:       c.IsActive,
		Metadata:       helpers.FromJSONB(c.Metadata),
		CreatedAt:      c.CreatedAt.Time,
		UpdatedAt:      c.UpdatedAt.Time,
	}
}
