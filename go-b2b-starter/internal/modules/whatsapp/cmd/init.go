package cmd

import (
	"fmt"

	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/infra/graphapi"
)

func Init(container *dig.Container) error {
	// Validated environment invariants: missing required Meta vars fail loudly
	// at startup so misconfiguration never reaches the signup flow.
	cfg, metaCfg, err := graphapi.FromEnv()
	if err != nil {
		return fmt.Errorf("whatsapp graph api config: %w", err)
	}

	if err := container.Provide(func() graphapi.Client {
		return graphapi.NewClient(cfg, nil)
	}); err != nil {
		return fmt.Errorf("failed to provide graph api client: %w", err)
	}
	if err := container.Provide(func() graphapi.MetaConfig {
		return metaCfg
	}); err != nil {
		return fmt.Errorf("failed to provide meta config: %w", err)
	}

	module := whatsapp.NewModule(container)
	if err := module.RegisterDependencies(); err != nil {
		return fmt.Errorf("failed to register whatsapp dependencies: %w", err)
	}

	provider := whatsapp.NewProvider(container)
	if err := provider.RegisterDependencies(); err != nil {
		return fmt.Errorf("failed to register whatsapp provider: %w", err)
	}

	return nil
}
