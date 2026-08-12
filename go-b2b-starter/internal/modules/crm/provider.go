package crm

import (
	"go.uber.org/dig"

	"github.com/jackc/pgx/v5/pgxpool"
	gen "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/auth/adapters/stytch"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/app/services"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
)

type Provider struct {
	container *dig.Container
}

func NewProvider(container *dig.Container) *Provider {
	return &Provider{container: container}
}

func (p *Provider) RegisterDependencies() error {
	if err := p.container.Provide(func(
		contactService services.ContactService,
		companyService services.CompanyService,
		dealService services.DealService,
		pipelineService services.PipelineService,
		activityService services.ActivityService,
		tagService services.TagService,
		conversationService services.ConversationService,
		outboundService services.OutboundService,
		sendGovernance services.ManualSendGovernance,
		featureProvider features.FeatureProvider,
		memberDirectory *stytch.MemberDirectoryService,
		logger logger.Logger,
	) *CRMHandler {
		return NewCRMHandler(contactService, companyService, dealService, pipelineService, activityService, tagService, conversationService, outboundService, sendGovernance, featureProvider, memberDirectory, logger)
	}); err != nil {
		return err
	}

	if err := p.container.Provide(func(
		handler *CRMHandler,
		featureProvider features.FeatureProvider,
		pool *pgxpool.Pool,
		store gen.Store,
		log logger.Logger,
	) *Routes {
		return NewRoutes(handler, featureProvider, pool, store, log)
	}); err != nil {
		return err
	}

	return nil
}
