package billing

import (
	"go.uber.org/dig"

	billingServices "github.com/moasq/go-b2b-starter/internal/modules/billing/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	mercadopagopkg "github.com/moasq/go-b2b-starter/internal/platform/mercadopago"
	polarpkg "github.com/moasq/go-b2b-starter/internal/platform/polar"
)

// handlerParams collects the dependencies for constructing the API handler.
// The MercadoPago config is optional: Polar-only deployments skip MP
// registration entirely, in which case the MP webhook secret is empty and MP
// webhooks are rejected as unconfigured.
type handlerParams struct {
	dig.In

	BillingService  billingServices.BillingService
	WebhookVerifier domain.WebhookVerifier
	PolarCfg        *polarpkg.Config
	MPCfg           *mercadopagopkg.Config `optional:"true"`
	Logger          logger.Logger
}

// RegisterHandlers registers subscription API handlers in the DI container
func RegisterHandlers(container *dig.Container) error {
	if err := container.Provide(func() domain.WebhookVerifier {
		return newWebhookVerifier()
	}); err != nil {
		return err
	}
	if err := container.Provide(func(p handlerParams) *Handler {
		mpSecret := ""
		if p.MPCfg != nil {
			mpSecret = p.MPCfg.WebhookSecret
		}
		return NewHandler(p.BillingService, p.WebhookVerifier, p.PolarCfg.WebhookSecret, mpSecret, p.Logger)
	}); err != nil {
		return err
	}
	return nil
}

// ProvideHandler is an alias for RegisterHandlers for consistency
func ProvideHandler(container *dig.Container) error {
	return RegisterHandlers(container)
}
