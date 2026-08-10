package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/campaigns/domain"
)

type campaignRepository struct {
	store sqlc.Store
}

func NewCampaignRepository(store sqlc.Store) domain.CampaignRepository {
	return &campaignRepository{store: store}
}

func (r *campaignRepository) Create(ctx context.Context, orgID int32, nombre string, segmentID int32, createdBy string) (*domain.Campaign, error) {
	result, err := r.store.CreateCampaign(ctx, sqlc.CreateCampaignParams{
		OrganizationID: orgID,
		Nombre:         nombre,
		SegmentID:      segmentID,
		CreatedBy:      pgText(createdBy),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create campaign: %w", err)
	}
	return mapCampaign(&result), nil
}

func (r *campaignRepository) Get(ctx context.Context, orgID, id int32) (*domain.Campaign, error) {
	result, err := r.store.GetCampaign(ctx, sqlc.GetCampaignParams{ID: id, OrganizationID: orgID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCampaignNotFound
		}
		return nil, fmt.Errorf("failed to get campaign: %w", err)
	}
	return mapCampaign(&result), nil
}

func (r *campaignRepository) List(ctx context.Context, orgID int32) ([]*domain.Campaign, error) {
	results, err := r.store.ListCampaigns(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list campaigns: %w", err)
	}
	campaigns := make([]*domain.Campaign, len(results))
	for i := range results {
		campaigns[i] = mapCampaign(&results[i])
	}
	return campaigns, nil
}

func (r *campaignRepository) Launch(ctx context.Context, orgID, id int32, recipientCount int32) (*domain.Campaign, error) {
	result, err := r.store.LaunchCampaign(ctx, sqlc.LaunchCampaignParams{
		ID:             id,
		OrganizationID: orgID,
		RecipientCount: recipientCount,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCampaignNotDraft
		}
		return nil, fmt.Errorf("failed to launch campaign: %w", err)
	}
	return mapCampaign(&result), nil
}

func (r *campaignRepository) SnapshotRecipients(ctx context.Context, campaignID int32, contactIDs []int32) (int64, error) {
	if len(contactIDs) == 0 {
		return 0, nil
	}
	rows, err := r.store.SnapshotCampaignRecipients(ctx, sqlc.SnapshotCampaignRecipientsParams{
		CampaignID: campaignID,
		Column2:    contactIDs,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to snapshot recipients: %w", err)
	}
	return rows, nil
}

func (r *campaignRepository) ListRecipients(ctx context.Context, campaignID int32, limit, offset int32) ([]*domain.CampaignRecipient, error) {
	results, err := r.store.ListCampaignRecipients(ctx, sqlc.ListCampaignRecipientsParams{
		CampaignID: campaignID,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list recipients: %w", err)
	}
	recipients := make([]*domain.CampaignRecipient, len(results))
	for i := range results {
		r := &results[i]
		recipients[i] = &domain.CampaignRecipient{
			ID:                r.ID,
			CampaignID:        r.CampaignID,
			ContactID:         r.ContactID,
			Status:            domain.RecipientStatus(r.Status),
			WhatsappMessageID: r.WhatsappMessageID.String,
			Error:             r.Error.String,
			PhoneNumber:       r.PhoneNumber.String,
			DisplayName:       r.DisplayName.String,
			CreatedAt:         r.CreatedAt.Time,
			UpdatedAt:         r.UpdatedAt.Time,
		}
	}
	return recipients, nil
}

func mapCampaign(c *sqlc.CrmCampaign) *domain.Campaign {
	campaign := &domain.Campaign{
		ID:             c.ID,
		OrganizationID: c.OrganizationID,
		Nombre:         c.Nombre,
		SegmentID:      c.SegmentID,
		Status:         domain.CampaignStatus(c.Status),
		RecipientCount: c.RecipientCount,
		CreatedBy:      c.CreatedBy.String,
		CreatedAt:      c.CreatedAt.Time,
		UpdatedAt:      c.UpdatedAt.Time,
	}
	if c.LaunchedAt.Valid {
		campaign.LaunchedAt = &c.LaunchedAt.Time
	}
	return campaign
}
