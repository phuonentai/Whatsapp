package cmd

import (
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/platform/mercadopago"
	"go.uber.org/dig"
)

func Init(container *dig.Container) error {
	if err := container.Provide(func() (*mercadopago.Config, error) {
		config, err := mercadopago.LoadConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load MercadoPago configuration: %w", err)
		}
		return &config, nil
	}); err != nil {
		return fmt.Errorf("failed to provide MercadoPago config: %w", err)
	}

	if err := mercadopago.Module(container); err != nil {
		return fmt.Errorf("failed to register MercadoPago module: %w", err)
	}

	return nil
}
