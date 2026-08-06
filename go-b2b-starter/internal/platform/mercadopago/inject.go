package mercadopago

import (
	"fmt"

	"go.uber.org/dig"
)

func Module(container *dig.Container) error {
	if err := container.Provide(NewClient); err != nil {
		return fmt.Errorf("failed to provide MercadoPago client: %w", err)
	}
	return nil
}
