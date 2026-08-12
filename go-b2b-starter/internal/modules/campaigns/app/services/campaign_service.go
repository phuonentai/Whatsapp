package services

import (
	"context"
	"fmt"
	"strings"

	crmDomain "github.com/moasq/go-b2b-starter/internal/modules/crm/domain"

	"github.com/moasq/go-b2b-starter/internal/modules/campaigns/domain"
)

type campaignService struct {
	campaignRepo domain.CampaignRepository
	segmentRepo  domain.SegmentRepository
	evaluator    domain.SegmentEvaluator
	activityRepo crmDomain.ActivityRepository
}

func NewCampaignService(
	campaignRepo domain.CampaignRepository,
	segmentRepo domain.SegmentRepository,
	evaluator domain.SegmentEvaluator,
	activityRepo crmDomain.ActivityRepository,
) CampaignService {
	return &campaignService{
		campaignRepo: campaignRepo,
		segmentRepo:  segmentRepo,
		evaluator:    evaluator,
		activityRepo: activityRepo,
	}
}

func (s *campaignService) Create(ctx context.Context, orgID int32, nombre string, segmentID int32, mensaje string, createdBy string) (*domain.Campaign, error) {
	if strings.TrimSpace(nombre) == "" {
		return nil, fmt.Errorf("%w: el nombre es obligatorio", domain.ErrInvalidFilterSpec)
	}
	// Verify the segment exists and belongs to the org before referencing it.
	if _, err := s.segmentRepo.Get(ctx, orgID, segmentID); err != nil {
		return nil, fmt.Errorf("segmento inválido: %w", err)
	}
	// Optional message body: trimmed; an empty value persists NULL (old
	// clients create null-message drafts). Nothing sends at create.
	return s.campaignRepo.Create(ctx, orgID, strings.TrimSpace(nombre), segmentID, strings.TrimSpace(mensaje), createdBy)
}

func (s *campaignService) Get(ctx context.Context, orgID, id int32) (*domain.Campaign, error) {
	return s.campaignRepo.Get(ctx, orgID, id)
}

func (s *campaignService) List(ctx context.Context, orgID int32) ([]*domain.Campaign, error) {
	return s.campaignRepo.List(ctx, orgID)
}

func (s *campaignService) Launch(ctx context.Context, orgID, id int32, createdBy string) (*domain.Campaign, error) {
	campaign, err := s.campaignRepo.Get(ctx, orgID, id)
	if err != nil {
		return nil, err
	}

	segment, err := s.segmentRepo.Get(ctx, orgID, campaign.SegmentID)
	if err != nil {
		return nil, err
	}

	// Evaluate once: filters + mandatory hard gates (consent, valid phone).
	contactIDs, err := s.evaluator.ContactIDs(ctx, orgID, segment.FilterSpec)
	if err != nil {
		return nil, err
	}

	inserted, err := s.campaignRepo.SnapshotRecipients(ctx, campaign.ID, contactIDs)
	if err != nil {
		return nil, err
	}

	// Guarded transition: single-row update WHERE status='draft' prevents
	// concurrent double snapshots.
	launched, err := s.campaignRepo.Launch(ctx, orgID, campaign.ID, int32(inserted))
	if err != nil {
		return nil, err
	}

	if err := s.auditLaunch(ctx, orgID, launched, createdBy); err != nil {
		return nil, err
	}

	return launched, nil
}

// auditLaunch records the launch in the CRM activity timeline (tipo=sistema).
func (s *campaignService) auditLaunch(ctx context.Context, orgID int32, campaign *domain.Campaign, createdBy string) error {
	_, err := s.activityRepo.Create(ctx, &crmDomain.Activity{
		OrganizationID: orgID,
		Tipo:           crmDomain.ActivityTypeSistema,
		Asunto:         "Campaña lanzada",
		Contenido:      campaign.Nombre,
		Metadata: map[string]interface{}{
			"campaign_id":     campaign.ID,
			"segment_id":      campaign.SegmentID,
			"recipient_count": campaign.RecipientCount,
			"member_id":       createdBy,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to audit campaign launch: %w", err)
	}
	return nil
}

func (s *campaignService) ListRecipients(ctx context.Context, orgID, campaignID int32, limit, offset int32) ([]*domain.CampaignRecipient, error) {
	if _, err := s.campaignRepo.Get(ctx, orgID, campaignID); err != nil {
		return nil, err
	}
	return s.campaignRepo.ListRecipients(ctx, campaignID, limit, offset)
}
