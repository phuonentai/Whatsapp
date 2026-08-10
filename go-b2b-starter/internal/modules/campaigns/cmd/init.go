package cmd

import (
	"fmt"

	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/campaigns"
)

func Init(container *dig.Container) error {
	module := campaigns.NewModule(container)
	if err := module.RegisterDependencies(); err != nil {
		return fmt.Errorf("failed to register campaigns dependencies: %w", err)
	}
	return nil
}
