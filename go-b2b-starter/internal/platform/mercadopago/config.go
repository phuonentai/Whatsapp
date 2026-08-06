package mercadopago

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	AccessToken    string `mapstructure:"MERCADOPAGO_ACCESS_TOKEN"`
	BaseURL        string `mapstructure:"MERCADOPAGO_BASE_URL"`
	WebhookSecret  string `mapstructure:"MERCADOPAGO_WEBHOOK_SECRET"`
	CheckoutPlanID string `mapstructure:"MERCADOPAGO_CHECKOUT_PLAN_ID"`
	BusinessPlanID string `mapstructure:"MERCADOPAGO_BUSINESS_PLAN_ID"`
	Debug          bool   `mapstructure:"MERCADOPAGO_DEBUG"`
}

func LoadConfig() (Config, error) {
	var cfg Config

	viper.SetConfigName("app")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	viper.SetDefault("MERCADOPAGO_BASE_URL", "https://api.mercadopago.com")
	viper.SetDefault("MERCADOPAGO_DEBUG", false)

	if err := viper.ReadInConfig(); err == nil {
		_ = err
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("unable to decode mercadopago config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.AccessToken == "" {
		return fmt.Errorf("mercadopago access token is required (MERCADOPAGO_ACCESS_TOKEN)")
	}
	if c.BaseURL == "" {
		return fmt.Errorf("mercadopago base URL is required (MERCADOPAGO_BASE_URL)")
	}
	return nil
}
