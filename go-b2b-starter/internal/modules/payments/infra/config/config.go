// Package config loads the client-payments environment configuration.
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	// CommissionRate is the platform commission markup (decimal, e.g. 0.025)
	// applied on top of client payment amounts for the MercadoPago rail.
	CommissionRate float64 `mapstructure:"PAYMENTS_COMMISSION_RATE"`
}

func LoadConfig() (Config, error) {
	var cfg Config

	viper.SetConfigName("app")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	viper.SetDefault("PAYMENTS_COMMISSION_RATE", 0.0)

	if err := viper.ReadInConfig(); err == nil {
		_ = err
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("unable to decode payments config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.CommissionRate < 0 || c.CommissionRate >= 1 {
		return fmt.Errorf("PAYMENTS_COMMISSION_RATE must be in [0, 1), got %f", c.CommissionRate)
	}
	return nil
}
