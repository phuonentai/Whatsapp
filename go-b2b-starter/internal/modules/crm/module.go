package crm

import (
	"github.com/moasq/go-b2b-starter/internal/modules/crm/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	whatsappDomain "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	"go.uber.org/dig"
)

type Module struct {
	container *dig.Container
}

func NewModule(container *dig.Container) *Module {
	return &Module{container: container}
}

func (m *Module) RegisterDependencies() error {
	if err := m.container.Provide(func(
		contactRepo domain.ContactRepository,
		featureProvider features.FeatureProvider,
	) services.ContactService {
		return services.NewContactService(contactRepo, featureProvider)
	}); err != nil {
		return err
	}

	if err := m.container.Provide(func(
		companyRepo domain.CompanyRepository,
		featureProvider features.FeatureProvider,
	) services.CompanyService {
		return services.NewCompanyService(companyRepo, featureProvider)
	}); err != nil {
		return err
	}

	if err := m.container.Provide(func(
		dealRepo domain.DealRepository,
		pipelineRepo domain.PipelineRepository,
		activityRepo domain.ActivityRepository,
		featureProvider features.FeatureProvider,
	) services.DealService {
		return services.NewDealService(dealRepo, pipelineRepo, activityRepo, featureProvider, nil)
	}); err != nil {
		return err
	}

	if err := m.container.Provide(func(
		pipelineRepo domain.PipelineRepository,
		stageRepo domain.PipelineStageRepository,
		featureProvider features.FeatureProvider,
	) services.PipelineService {
		return services.NewPipelineService(pipelineRepo, stageRepo, featureProvider)
	}); err != nil {
		return err
	}

	if err := m.container.Provide(func(
		activityRepo domain.ActivityRepository,
		featureProvider features.FeatureProvider,
	) services.ActivityService {
		return services.NewActivityService(activityRepo, featureProvider)
	}); err != nil {
		return err
	}

	if err := m.container.Provide(func(
		tagRepo domain.TagRepository,
		entityTagRepo domain.EntityTagRepository,
		featureProvider features.FeatureProvider,
	) services.TagService {
		return services.NewTagService(tagRepo, entityTagRepo, featureProvider)
	}); err != nil {
		return err
	}

	if err := m.container.Provide(func(
		convRepo domain.ConversationRepository,
		msgRepo domain.MessageRepository,
		contactRepo domain.ContactRepository,
	) services.ConversationService {
		return services.NewConversationService(convRepo, msgRepo, contactRepo)
	}); err != nil {
		return err
	}

	if err := m.container.Provide(func(
		contactRepo domain.ContactRepository,
		conversationRepo domain.ConversationRepository,
		messageRepo domain.MessageRepository,
		activityRepo domain.ActivityRepository,
		featureProvider features.FeatureProvider,
		log logger.Logger,
	) services.CRMService {
		return services.NewCRMService(contactRepo, conversationRepo, messageRepo, activityRepo, featureProvider, log)
	}); err != nil {
		return err
	}

	if err := m.container.Provide(func(
		crmService services.CRMService,
		log logger.Logger,
	) services.MessageListener {
		return services.NewMessageListener(crmService, log)
	}); err != nil {
		return err
	}

	if err := m.container.Provide(func(
		convRepo domain.ConversationRepository,
		contactRepo domain.ContactRepository,
		msgRepo domain.MessageRepository,
		whatsappRepo whatsappDomain.ConfigRepository,
	) services.OutboundService {
		return services.NewOutboundService(convRepo, contactRepo, msgRepo, whatsappRepo)
	}); err != nil {
		return err
	}

	return nil
}
