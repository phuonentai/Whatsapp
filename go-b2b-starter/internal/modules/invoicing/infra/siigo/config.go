// Package siigo implements the Siigo electronic-invoicing adapter.
//
// Siigo API specifics are unverified (no network access in this environment);
// the adapter follows the assumed OAuth2 client_credentials + REST contract
// from the change proposal. Endpoint paths are configurable so a verification
// spike at deployment can adjust them without touching the domain or service.
package siigo

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	ClientID      string `mapstructure:"SIIGO_CLIENT_ID"`
	ClientSecret  string `mapstructure:"SIIGO_CLIENT_SECRET"`
	BaseURL       string `mapstructure:"SIIGO_BASE_URL"`
	TokenURL      string `mapstructure:"SIIGO_TOKEN_URL"`
	WebhookSecret string `mapstructure:"SIIGO_WEBHOOK_SECRET"`
	Sandbox       bool   `mapstructure:"SIIGO_SANDBOX"`
	Debug         bool   `mapstructure:"SIIGO_DEBUG"`
}

// LoadConfig reads the Siigo config from env. Sandbox is the default so no
// live calls happen until explicitly configured for production.
func LoadConfig() (Config, error) {
	var cfg Config

	viper.SetConfigName("app")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	viper.SetDefault("SIIGO_BASE_URL", "https://api.siigo.com")
	viper.SetDefault("SIIGO_TOKEN_URL", "https://siigo.com/token")
	viper.SetDefault("SIIGO_SANDBOX", true)
	viper.SetDefault("SIIGO_DEBUG", false)
	viper.SetDefault("SIIGO_WEBHOOK_SECRET", "")

	if err := viper.ReadInConfig(); err == nil {
		_ = err
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("unable to decode siigo config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if c.ClientID == "" {
		return fmt.Errorf("siigo client id is required (SIIGO_CLIENT_ID)")
	}
	if c.ClientSecret == "" {
		return fmt.Errorf("siigo client secret is required (SIIGO_CLIENT_SECRET)")
	}
	if c.BaseURL == "" {
		return fmt.Errorf("siigo base URL is required (SIIGO_BASE_URL)")
	}
	return nil
}

// TokenTTL bounds the in-memory access-token cache (mirrors the JWKS cache
// convention). Adapter refreshes after this window.
const TokenTTL = 300 * time.Second
