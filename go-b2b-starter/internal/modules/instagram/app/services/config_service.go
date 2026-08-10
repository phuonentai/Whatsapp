package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/moasq/go-b2b-starter/internal/modules/instagram/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/instagram/infra/graphapi"
)

type ConfigService interface {
	GetConfig(ctx context.Context, orgID int32) (*domain.InstagramConfig, error)
	UpsertConfig(ctx context.Context, orgID int32, input *domain.InstagramConfig) (*domain.InstagramConfig, error)
	ToggleConfig(ctx context.Context, orgID int32) (*domain.InstagramConfig, error)
	RefreshToken(ctx context.Context, orgID int32, appID, appSecret string) (*domain.InstagramConfig, error)
}

type configService struct {
	configRepo domain.ConfigRepository
	client     graphapi.IGClient
}

func NewConfigService(configRepo domain.ConfigRepository, client graphapi.IGClient) ConfigService {
	return &configService{configRepo: configRepo, client: client}
}

func (s *configService) GetConfig(ctx context.Context, orgID int32) (*domain.InstagramConfig, error) {
	config, err := s.configRepo.GetByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrConfigNotFound, err)
	}
	return maskConfig(config), nil
}

func (s *configService) UpsertConfig(ctx context.Context, orgID int32, input *domain.InstagramConfig) (*domain.InstagramConfig, error) {
	existing, err := s.configRepo.GetByOrganizationID(ctx, orgID)
	if err != nil {
		if errors.Is(err, domain.ErrConfigNotFound) {
			return s.createConfig(ctx, orgID, input)
		}
		return nil, fmt.Errorf("failed to check existing config: %w", err)
	}
	return s.updateConfig(ctx, existing, input)
}

func (s *configService) ToggleConfig(ctx context.Context, orgID int32) (*domain.InstagramConfig, error) {
	config, err := s.configRepo.GetByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrConfigNotFound, err)
	}

	config.IsActive = !config.IsActive

	updated, err := s.configRepo.Update(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to toggle config: %w", err)
	}
	return maskConfig(updated), nil
}

// RefreshToken exchanges the stored access token for a new long-lived token
// via the Meta fb_exchange_token grant. On failure the stored token is left
// untouched.
func (s *configService) RefreshToken(ctx context.Context, orgID int32, appID, appSecret string) (*domain.InstagramConfig, error) {
	config, err := s.configRepo.GetByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrConfigNotFound, err)
	}
	if config.AccessToken == "" {
		return nil, fmt.Errorf("%w: access token is missing", domain.ErrTokenRefreshFailed)
	}

	newToken, expiry, err := s.client.RefreshToken(ctx, appID, appSecret, config.GraphAPIURL, config.APIVersion, config.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrTokenRefreshFailed, err)
	}

	config.AccessToken = newToken
	config.TokenExpiresAt = expiry

	updated, err := s.configRepo.Update(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to persist refreshed token: %w", err)
	}
	return maskConfig(updated), nil
}

func (s *configService) createConfig(ctx context.Context, orgID int32, input *domain.InstagramConfig) (*domain.InstagramConfig, error) {
	input.OrganizationID = orgID

	if err := input.Validate(); err != nil {
		return nil, err
	}
	if input.AccessToken == "" {
		return nil, fmt.Errorf("access token is required")
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
			return nil, fmt.Errorf("%w: this IG user ID is already in use by another organization", domain.ErrIGUserIDConflict)
		}
		return nil, fmt.Errorf("failed to create config: %w", err)
	}
	return maskConfig(config), nil
}

func (s *configService) updateConfig(ctx context.Context, existing *domain.InstagramConfig, input *domain.InstagramConfig) (*domain.InstagramConfig, error) {
	if input.IGUserID != "" {
		existing.IGUserID = input.IGUserID
	}
	if input.IGUsername != "" {
		existing.IGUsername = input.IGUsername
	}
	if input.FBPageID != "" {
		existing.FBPageID = input.FBPageID
	}
	if input.APIVersion != "" {
		existing.APIVersion = input.APIVersion
	}
	if input.GraphAPIURL != "" {
		existing.GraphAPIURL = input.GraphAPIURL
	}
	if input.TokenExpiresAt != nil {
		existing.TokenExpiresAt = input.TokenExpiresAt
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
		if isDuplicateKeyError(err) {
			return nil, fmt.Errorf("%w: this IG user ID is already in use by another organization", domain.ErrIGUserIDConflict)
		}
		return nil, fmt.Errorf("failed to update config: %w", err)
	}
	return maskConfig(updated), nil
}

// maskConfig redacts secrets before the config leaves the service.
func maskConfig(c *domain.InstagramConfig) *domain.InstagramConfig {
	if c == nil {
		return nil
	}
	cloned := *c
	cloned.WebhookSecret = MaskSecret(cloned.WebhookSecret)
	cloned.VerifyToken = MaskSecret(cloned.VerifyToken)
	cloned.AccessToken = MaskSecret(cloned.AccessToken)
	return &cloned
}

func MaskSecret(s string) string {
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
