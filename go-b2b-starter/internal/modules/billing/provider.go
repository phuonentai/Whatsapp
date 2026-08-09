package billing

import (
	"go.uber.org/dig"

	billingServices "github.com/moasq/go-b2b-starter/internal/modules/billing/app/services"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	mercadopagopkg "github.com/moasq/go-b2b-starter/internal/platform/mercadopago"
	polarpkg "github.com/moasq/go-b2b-starter/internal/platform/polar"
)

// RegisterHandlers registers subscription API handlers in the DI container
func RegisterHandlers(container *dig.Container) error {
	if err := container.Provide(func(
		billingService billingServices.BillingService,
		polarCfg *polarpkg.Config,
		mpCfg *mercadopagopkg.Config,
		log logger.Logger,
	) *Handler {
		return NewHandler(billingService, polarCfg.WebhookSecret, mpCfg.WebhookSecret, log)
	}); err != nil {
		return err
	}
	return nil
}

// ProvideHandler is an alias for RegisterHandlers for consistency
func ProvideHandler(container *dig.Container) error {
	return RegisterHandlers(container)
}
