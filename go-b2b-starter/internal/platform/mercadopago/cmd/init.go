package cmd

import (
	"fmt"
	"os"

	"github.com/moasq/go-b2b-starter/internal/platform/mercadopago"
	"go.uber.org/dig"
)

// Init registers the MercadoPago config and client when credentials are
// configured. When MERCADOPAGO_ACCESS_TOKEN is unset the backend boots
// Polar-only: no config/client is registered, the billing DI container treats
// the named "mercadopago" adapter as optional, and MP service calls return a
// clear "mercadopago not configured" error.
func Init(container *dig.Container) error {
	if os.Getenv("MERCADOPAGO_ACCESS_TOKEN") == "" {
		return nil
	}

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
