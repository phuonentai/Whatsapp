package cmd

import (
	"fmt"

	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp"
)

func Init(container *dig.Container) error {
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
