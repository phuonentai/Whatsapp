package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	whatsappDomain "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/infra/graphapi"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// TemplateService is the application boundary for the org-scoped template
// registry. It owns the local lifecycle and coordinates Meta submission and
// status reconciliation through the graphapi client (org credentials resolved
// from whatsapp_configs; never stored on template rows).
type TemplateService interface {
	CreateTemplate(ctx context.Context, orgID int32, input *TemplateInput) (*whatsappDomain.Template, error)
	UpdateTemplate(ctx context.Context, orgID int32, id int64, input *TemplateInput) (*whatsappDomain.Template, error)
	DeleteTemplate(ctx context.Context, orgID int32, id int64) error
	ListTemplates(ctx context.Context, orgID int32) ([]*whatsappDomain.Template, error)
	GetTemplate(ctx context.Context, orgID int32, id int64) (*whatsappDomain.Template, error)
	SubmitTemplate(ctx context.Context, orgID int32, id int64) (*whatsappDomain.Template, error)
	RefreshTemplateStatus(ctx context.Context, orgID int32, id int64) (*whatsappDomain.Template, error)
}

// TemplateInput carries the editable authoring fields.
type TemplateInput struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Language string `json:"language"`
	Body     string `json:"body"`
}

type templateService struct {
	repo       whatsappDomain.TemplateRepository
	configRepo whatsappDomain.ConfigRepository
	graph      graphapi.Client
	logger     logger.Logger
}

func NewTemplateService(
	repo whatsappDomain.TemplateRepository,
	configRepo whatsappDomain.ConfigRepository,
	graph graphapi.Client,
	log logger.Logger,
) TemplateService {
	return &templateService{
		repo:       repo,
		configRepo: configRepo,
		graph:      graph,
		logger:     log,
	}
}

func (s *templateService) CreateTemplate(ctx context.Context, orgID int32, input *TemplateInput) (*whatsappDomain.Template, error) {
	t := &whatsappDomain.Template{
		OrganizationID: orgID,
		Name:           strings.TrimSpace(input.Name),
		Category:       strings.TrimSpace(input.Category),
		Language:       strings.TrimSpace(input.Language),
		Body:           strings.TrimSpace(input.Body),
		Status:         whatsappDomain.TemplateStatusDraft,
		IsActive:       true,
	}
	t.ParamCount = whatsappDomain.CountParams(t.Body)

	if err := t.Validate(); err != nil {
		return nil, err
	}

	created, err := s.repo.Create(ctx, t)
	if err != nil {
		if errors.Is(err, whatsappDomain.ErrTemplateNameConflict) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to create template: %w", err)
	}
	return created, nil
}

func (s *templateService) UpdateTemplate(ctx context.Context, orgID int32, id int64, input *TemplateInput) (*whatsappDomain.Template, error) {
	existing, err := s.repo.GetByID(ctx, orgID, id)
	if err != nil {
		return nil, err
	}
	if !existing.IsEditable() {
		return nil, whatsappDomain.ErrTemplateNotDraft
	}

	existing.Name = strings.TrimSpace(input.Name)
	existing.Category = strings.TrimSpace(input.Category)
	existing.Language = strings.TrimSpace(input.Language)
	existing.Body = strings.TrimSpace(input.Body)
	existing.ParamCount = whatsappDomain.CountParams(existing.Body)

	if err := existing.Validate(); err != nil {
		return nil, err
	}

	updated, err := s.repo.Update(ctx, existing)
	if err != nil {
		if errors.Is(err, whatsappDomain.ErrTemplateNameConflict) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to update template: %w", err)
	}
	return updated, nil
}

func (s *templateService) DeleteTemplate(ctx context.Context, orgID int32, id int64) error {
	existing, err := s.repo.GetByID(ctx, orgID, id)
	if err != nil {
		return err
	}
	if !existing.IsEditable() {
		return whatsappDomain.ErrTemplateNotDraft
	}
	return s.repo.Delete(ctx, orgID, id)
}

func (s *templateService) ListTemplates(ctx context.Context, orgID int32) ([]*whatsappDomain.Template, error) {
	return s.repo.ListByOrg(ctx, orgID)
}

func (s *templateService) GetTemplate(ctx context.Context, orgID int32, id int64) (*whatsappDomain.Template, error) {
	return s.repo.GetByID(ctx, orgID, id)
}

// SubmitTemplate pushes a draft to Meta and records the meta_template_id.
// Idempotent: a template already in submitted status returns its current
// state without a second Meta call. Templates in rejected status may be
// re-submitted directly (editing first returns them to draft).
func (s *templateService) SubmitTemplate(ctx context.Context, orgID int32, id int64) (*whatsappDomain.Template, error) {
	t, err := s.repo.GetByID(ctx, orgID, id)
	if err != nil {
		return nil, err
	}

	switch t.Status {
	case whatsappDomain.TemplateStatusSubmitted, whatsappDomain.TemplateStatusApproved, whatsappDomain.TemplateStatusPaused:
		// Idempotent: already at or beyond submitted — return current state
		// without contacting Meta.
		return t, nil
	case whatsappDomain.TemplateStatusDraft, whatsappDomain.TemplateStatusRejected:
		// proceed
	default:
		return nil, fmt.Errorf("%w: %s", whatsappDomain.ErrTemplateInvalidTransition, t.Status)
	}

	if err := t.Validate(); err != nil {
		return nil, err
	}

	config, err := s.configRepo.GetByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("whatsapp_not_configured: %w", err)
	}
	if !config.IsActive {
		return nil, fmt.Errorf("whatsapp_not_configured: config is inactive")
	}
	if config.AccessToken == "" {
		return nil, fmt.Errorf("whatsapp_not_configured: access token is missing")
	}
	if config.PhoneNumberID == "" {
		return nil, fmt.Errorf("whatsapp_not_configured: phone number id is missing")
	}

	apiVersion := config.APIVersion
	if apiVersion == "" {
		apiVersion = "v21.0"
	}
	graphAPIURL := config.GraphAPIURL
	if graphAPIURL == "" {
		graphAPIURL = "https://graph.facebook.com"
	}

	metaID, err := s.graph.SubmitTemplate(
		ctx,
		config.AccessToken,
		graphAPIURL,
		apiVersion,
		config.PhoneNumberID,
		t.Name,
		t.Language,
		t.Category,
		t.Body,
	)
	if err != nil {
		// Local state unchanged on Meta failure.
		return nil, fmt.Errorf("whatsapp_api_error: %w", err)
	}

	updated, err := s.repo.UpdateStatus(ctx, orgID, id, whatsappDomain.TemplateStatusSubmitted, &metaID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to record submission: %w", err)
	}
	if updated == nil {
		// Concurrent state change; return the freshly loaded template.
		return s.repo.GetByID(ctx, orgID, id)
	}

	s.logger.Info("template submitted to Meta", loggerdomain.Fields{
		"organization_id": orgID,
		"template_id":     id,
		"meta_template_id": metaID,
	})
	return updated, nil
}

// RefreshTemplateStatus reconciles the local status with Meta's current
// approval state. Idempotent: matching status is a no-op.
func (s *templateService) RefreshTemplateStatus(ctx context.Context, orgID int32, id int64) (*whatsappDomain.Template, error) {
	t, err := s.repo.GetByID(ctx, orgID, id)
	if err != nil {
		return nil, err
	}
	if t.MetaTemplateID == nil || *t.MetaTemplateID == "" {
		return nil, whatsappDomain.ErrTemplateNotFoundAtMeta
	}

	config, err := s.configRepo.GetByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("whatsapp_not_configured: %w", err)
	}
	if !config.IsActive || config.AccessToken == "" || config.PhoneNumberID == "" {
		return nil, fmt.Errorf("whatsapp_not_configured: config missing or inactive")
	}

	apiVersion := config.APIVersion
	if apiVersion == "" {
		apiVersion = "v21.0"
	}
	graphAPIURL := config.GraphAPIURL
	if graphAPIURL == "" {
		graphAPIURL = "https://graph.facebook.com"
	}

	metaStatus, err := s.graph.GetTemplateStatus(
		ctx,
		config.AccessToken,
		graphAPIURL,
		apiVersion,
		config.PhoneNumberID,
		*t.MetaTemplateID,
	)
	if err != nil {
		var gErr *graphapi.GraphError
		if errors.As(err, &gErr) && gErr.Code == 100 {
			// Template no longer exists at Meta.
			return nil, whatsappDomain.ErrTemplateNotFoundAtMeta
		}
		return nil, fmt.Errorf("whatsapp_api_error: %w", err)
	}

	localStatus := mapMetaStatus(metaStatus)
	if localStatus == "" {
		// Meta returned an unrecognized status; keep local state unchanged.
		s.logger.Warn("unrecognized meta template status", loggerdomain.Fields{
			"organization_id": orgID,
			"template_id":     id,
			"meta_status":     metaStatus,
		})
		return t, nil
	}

	updated, err := s.repo.UpdateStatus(ctx, orgID, id, localStatus, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to record refreshed status: %w", err)
	}
	if updated == nil {
		return t, nil // no-op: status unchanged
	}
	return updated, nil
}

// mapMetaStatus converts Meta's approval status vocabulary to the local one.
// Unknown statuses map to "" (caller keeps local state unchanged).
func mapMetaStatus(metaStatus string) whatsappDomain.TemplateStatus {
	switch strings.ToUpper(metaStatus) {
	case "APPROVED":
		return whatsappDomain.TemplateStatusApproved
	case "REJECTED":
		return whatsappDomain.TemplateStatusRejected
	case "PAUSED", "DISABLED":
		return whatsappDomain.TemplateStatusPaused
	case "PENDING", "IN_APPEAL", "IN_FLIGHT", "DELETED":
		return whatsappDomain.TemplateStatusSubmitted
	default:
		return ""
	}
}
