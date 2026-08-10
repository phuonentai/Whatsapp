package services

import (
	"context"
	"fmt"

	crmdomain "github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	igDomain "github.com/moasq/go-b2b-starter/internal/modules/instagram/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/instagram/domain/events"
	"github.com/moasq/go-b2b-starter/internal/modules/instagram/infra/graphapi"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// ProfileBackfillListener consumes instagram.profile.backfill events and
// resolves the contact's username/avatar from the Graph API using the
// organization's stored IG token. Transient failures bubble up so the outbox
// retries with backoff; permanent failures dead-letter.
type ProfileBackfillListener struct {
	configRepo  igDomain.ConfigRepository
	contactRepo crmdomain.ContactRepository
	client      graphapi.IGClient
	logger      logger.Logger
}

func NewProfileBackfillListener(
	configRepo igDomain.ConfigRepository,
	contactRepo crmdomain.ContactRepository,
	client graphapi.IGClient,
	log logger.Logger,
) *ProfileBackfillListener {
	return &ProfileBackfillListener{
		configRepo:  configRepo,
		contactRepo: contactRepo,
		client:      client,
		logger:      log,
	}
}

func (l *ProfileBackfillListener) Handle(ctx context.Context, event interface{}) error {
	backfill, ok := event.(*events.ProfileBackfill)
	if !ok {
		return fmt.Errorf("unexpected event type %T", event)
	}

	config, err := l.configRepo.GetByOrganizationID(ctx, backfill.OrganizationID)
	if err != nil {
		return fmt.Errorf("resolve instagram config for backfill: %w", err)
	}
	if !config.IsActive || config.AccessToken == "" {
		return fmt.Errorf("instagram config inactive or missing token for org %d", backfill.OrganizationID)
	}

	user, err := l.client.GetIGUser(ctx, config.AccessToken, config.GraphAPIURL, config.APIVersion, backfill.IGUserID)
	if err != nil {
		return fmt.Errorf("resolve ig user %s: %w", backfill.IGUserID, err)
	}

	displayName := user.Username
	if displayName == "" {
		displayName = backfill.IGUserID
	}

	if _, err := l.contactRepo.UpdateInstagramProfile(
		ctx,
		backfill.OrganizationID,
		backfill.ContactID,
		user.Username,
		user.ProfilePictureURL,
		displayName,
	); err != nil {
		return fmt.Errorf("persist instagram profile: %w", err)
	}

	l.logger.Info("instagram profile backfilled", loggerdomain.Fields{
		"org_id":      backfill.OrganizationID,
		"contact_id":  backfill.ContactID,
		"ig_user_id":  backfill.IGUserID,
		"ig_username": user.Username,
	})
	return nil
}
