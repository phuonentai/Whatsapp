// Package cmd wires the payments module into the dependency container.
package cmd

import (
	"fmt"

	"go.uber.org/dig"

	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	invoicingServices "github.com/moasq/go-b2b-starter/internal/modules/invoicing/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/payments/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/payments/domain"
	paymentsconfig "github.com/moasq/go-b2b-starter/internal/modules/payments/infra/config"
	paymentsMP "github.com/moasq/go-b2b-starter/internal/modules/payments/infra/mercadopago"
	"github.com/moasq/go-b2b-starter/internal/modules/payments/infra/repositories"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
	mp "github.com/moasq/go-b2b-starter/internal/platform/mercadopago"
)

func Init(container *dig.Container) error {
	if err := ProvideDependencies(container); err != nil {
		return fmt.Errorf("failed to provide payments dependencies: %w", err)
	}
	return nil
}

func ProvideDependencies(container *dig.Container) error {
	cfg, err := paymentsconfig.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load payments config: %w", err)
	}
	if err := container.Provide(func() float64 { return cfg.CommissionRate }, dig.Name("payments_commission_rate")); err != nil {
		return err
	}

	if err := container.Provide(func(store sqlc.Store) domain.PaymentRepository {
		return repositories.NewPaymentRepository(store)
	}); err != nil {
		return fmt.Errorf("failed to provide payment repository: %w", err)
	}

	if err := container.Provide(func(client *mp.Client, mpCfg *mp.Config, log loggerDomain.Logger) domain.PaymentRail {
		return paymentsMP.NewPaymentRail(client, log, mpCfg.BackURL)
	}); err != nil {
		return fmt.Errorf("failed to provide payment rail: %w", err)
	}

	if err := services.NewModule().Configure(container); err != nil {
		return fmt.Errorf("failed to configure payments services: %w", err)
	}

	// Real PaymentLinker for the invoicing module (replaces the noop seam):
	// the invoice WhatsApp notification carries a tracked payment link.
	if err := container.Provide(func(svc services.PaymentsService) invoicingServices.PaymentLinker {
		return services.NewInvoicingPaymentLinker(svc)
	}); err != nil {
		return fmt.Errorf("failed to provide invoicing payment linker: %w", err)
	}

	return nil
}
