package analytics

import (
	"go.uber.org/dig"

	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	analyticsServices "github.com/moasq/go-b2b-starter/internal/modules/analytics/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/analytics/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/analytics/infra/repositories"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
)

type Provider struct {
	container *dig.Container
}

func NewProvider(container *dig.Container) *Provider {
	return &Provider{container: container}
}

func (p *Provider) RegisterDependencies() error {
	if err := p.container.Provide(func(store sqlc.Store) domain.AnalyticsRepository {
		return repositories.NewAnalyticsRepository(store)
	}); err != nil {
		return err
	}

	if err := p.container.Provide(func(
		repo domain.AnalyticsRepository,
	) *analyticsServices.SalesReportService {
		return analyticsServices.NewSalesReportService(repo)
	}); err != nil {
		return err
	}

	if err := p.container.Provide(func(
		reportService *analyticsServices.SalesReportService,
	) *Handler {
		return NewHandler(reportService)
	}); err != nil {
		return err
	}

	if err := p.container.Provide(func(
		handler *Handler,
		featureProvider features.FeatureProvider,
	) *Routes {
		return NewRoutes(handler, featureProvider)
	}); err != nil {
		return err
	}

	return nil
}
