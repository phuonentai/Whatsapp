package crm

import (
	"go.uber.org/dig"

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
		featureProvider features.FeatureProvider,
		logger logger.Logger,
	) *CRMHandler {
		return NewCRMHandler(contactService, companyService, dealService, pipelineService, activityService, tagService, conversationService, outboundService, featureProvider, logger)
	}); err != nil {
		return err
	}

	if err := p.container.Provide(func(handler *CRMHandler, featureProvider features.FeatureProvider) *Routes {
		return NewRoutes(handler, featureProvider)
	}); err != nil {
		return err
	}

	return nil
}
