package campaigns

import (
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/campaigns/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/campaigns/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/campaigns/infra/repositories"
	billingServices "github.com/moasq/go-b2b-starter/internal/modules/billing/app/services"
	crmDomain "github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	llmdomain "github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
	"go.uber.org/dig"
)

type Module struct {
	container *dig.Container
}

func NewModule(container *dig.Container) *Module {
	return &Module{container: container}
}

func (m *Module) RegisterDependencies() error {
	// Infrastructure adapters.
	if err := m.container.Provide(func(store sqlc.Store) domain.SegmentRepository {
		return repositories.NewSegmentRepository(store)
	}); err != nil {
		return err
	}
	if err := m.container.Provide(func(store sqlc.Store) domain.CampaignRepository {
		return repositories.NewCampaignRepository(store)
	}); err != nil {
		return err
	}
	if err := m.container.Provide(func(store sqlc.Store) domain.SegmentEvaluator {
		return repositories.NewSegmentEvaluator(store)
	}); err != nil {
		return err
	}

	// Application services.
	if err := m.container.Provide(func(
		segmentRepo domain.SegmentRepository,
		evaluator domain.SegmentEvaluator,
		tagRepo crmDomain.TagRepository,
	) services.SegmentService {
		return services.NewSegmentService(segmentRepo, evaluator, tagRepo)
	}); err != nil {
		return err
	}
	if err := m.container.Provide(func(
		campaignRepo domain.CampaignRepository,
		segmentRepo domain.SegmentRepository,
		evaluator domain.SegmentEvaluator,
		activityRepo crmDomain.ActivityRepository,
	) services.CampaignService {
		return services.NewCampaignService(campaignRepo, segmentRepo, evaluator, activityRepo)
	}); err != nil {
		return err
	}
	if err := m.container.Provide(func(
		llm llmdomain.LLMClient,
		billing billingServices.BillingService,
		tagRepo crmDomain.TagRepository,
		evaluator domain.SegmentEvaluator,
	) services.AudienceBuilderService {
		return services.NewAudienceBuilder(llm, billing, tagRepo, evaluator)
	}); err != nil {
		return err
	}

	// HTTP layer.
	if err := m.container.Provide(func(
		segmentService services.SegmentService,
		campaignService services.CampaignService,
		aiBuilder services.AudienceBuilderService,
	) *Handler {
		return NewHandler(segmentService, campaignService, aiBuilder)
	}); err != nil {
		return err
	}
	if err := m.container.Provide(func(handler *Handler) *Routes {
		return NewRoutes(handler)
	}); err != nil {
		return err
	}

	return nil
}
