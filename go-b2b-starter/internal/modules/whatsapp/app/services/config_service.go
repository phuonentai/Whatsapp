package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
)

type ConfigService interface {
	GetConfig(ctx context.Context, orgID int32) (*domain.WhatsAppConfig, error)
	UpsertConfig(ctx context.Context, orgID int32, input *domain.WhatsAppConfig) (*domain.WhatsAppConfig, error)
	ToggleConfig(ctx context.Context, orgID int32) (*domain.WhatsAppConfig, error)
}

type configService struct {
	configRepo domain.ConfigRepository
}

func NewConfigService(configRepo domain.ConfigRepository) ConfigService {
	return &configService{configRepo: configRepo}
}

func (s *configService) GetConfig(ctx context.Context, orgID int32) (*domain.WhatsAppConfig, error) {
	config, err := s.configRepo.GetByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrConfigNotFound, err)
	}

	config.WebhookSecret = maskSecret(config.WebhookSecret)
	config.VerifyToken = maskSecret(config.VerifyToken)
	config.AccessToken = maskSecret(config.AccessToken)

	return config, nil
}

func (s *configService) UpsertConfig(ctx context.Context, orgID int32, input *domain.WhatsAppConfig) (*domain.WhatsAppConfig, error) {
	existing, err := s.configRepo.GetByOrganizationID(ctx, orgID)
	if err != nil {
		if errors.Is(err, domain.ErrConfigNotFound) {
			return s.createConfig(ctx, orgID, input)
		}
		return nil, fmt.Errorf("failed to check existing config: %w", err)
	}

	return s.updateConfig(ctx, existing, input)
}

func (s *configService) ToggleConfig(ctx context.Context, orgID int32) (*domain.WhatsAppConfig, error) {
	config, err := s.configRepo.GetByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrConfigNotFound, err)
	}

	config.IsActive = !config.IsActive

	updated, err := s.configRepo.Update(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to toggle config: %w", err)
	}

	updated.WebhookSecret = maskSecret(updated.WebhookSecret)
	updated.VerifyToken = maskSecret(updated.VerifyToken)
	updated.AccessToken = maskSecret(updated.AccessToken)

	return updated, nil
}

func (s *configService) createConfig(ctx context.Context, orgID int32, input *domain.WhatsAppConfig) (*domain.WhatsAppConfig, error) {
	input.OrganizationID = orgID

	if err := input.Validate(); err != nil {
		return nil, err
	}

	if input.BusinessPhone == "" {
		return nil, fmt.Errorf("business phone is required")
	}

	if input.APIVersion == "" {
		input.APIVersion = "v21.0"
	}
	if input.GraphAPIURL == "" {
		input.GraphAPIURL = "https://graph.facebook.com"
	}

	config, err := s.configRepo.Create(ctx, input)
	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, fmt.Errorf("phone_number_id_conflict: this phone number ID is already in use by another organization")
		}
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	config.WebhookSecret = maskSecret(config.WebhookSecret)
	config.VerifyToken = maskSecret(config.VerifyToken)
	config.AccessToken = maskSecret(config.AccessToken)

	return config, nil
}

func (s *configService) updateConfig(ctx context.Context, existing *domain.WhatsAppConfig, input *domain.WhatsAppConfig) (*domain.WhatsAppConfig, error) {
	if input.PhoneNumberID != "" {
		existing.PhoneNumberID = input.PhoneNumberID
	}
	if input.BusinessPhone != "" {
		existing.BusinessPhone = input.BusinessPhone
	}
	if input.AppID != "" {
		existing.AppID = input.AppID
	}
	if input.WABAID != "" {
		existing.WABAID = input.WABAID
	}
	if input.APIVersion != "" {
		existing.APIVersion = input.APIVersion
	}
	if input.GraphAPIURL != "" {
		existing.GraphAPIURL = input.GraphAPIURL
	}

	if input.WebhookSecret != "" && !isMasked(input.WebhookSecret) {
		existing.WebhookSecret = input.WebhookSecret
	}
	if input.VerifyToken != "" && !isMasked(input.VerifyToken) {
		existing.VerifyToken = input.VerifyToken
	}
	if input.AccessToken != "" && !isMasked(input.AccessToken) {
		existing.AccessToken = input.AccessToken
	}

	if input.Metadata != nil {
		existing.Metadata = input.Metadata
	}

	updated, err := s.configRepo.Update(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	updated.WebhookSecret = maskSecret(updated.WebhookSecret)
	updated.VerifyToken = maskSecret(updated.VerifyToken)
	updated.AccessToken = maskSecret(updated.AccessToken)

	return updated, nil
}

func maskSecret(s string) string {
	if len(s) <= 6 {
		return "****"
	}
	return s[:6] + "****" + s[len(s)-4:]
}

func isMasked(s string) bool {
	return strings.Contains(s, "****")
}

func isDuplicateKeyError(err error) bool {
	return strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "unique") ||
		strings.Contains(err.Error(), "violates unique constraint")
}
