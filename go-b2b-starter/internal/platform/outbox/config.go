package outbox

import (
	"time"

	"github.com/spf13/viper"
)

// Config holds outbox dispatcher tuning.
type Config struct {
	PollInterval  time.Duration `mapstructure:"OUTBOX_POLL_INTERVAL"`
	MaxAttempts   int           `mapstructure:"OUTBOX_MAX_ATTEMPTS"`
	BatchSize     int32         `mapstructure:"OUTBOX_BATCH_SIZE"`
	Enabled       bool          `mapstructure:"OUTBOX_DISPATCHER_ENABLED"`
	BackoffBase   time.Duration `mapstructure:"OUTBOX_BACKOFF_BASE"`
	BackoffMax    time.Duration `mapstructure:"OUTBOX_BACKOFF_MAX"`
	RetentionDays int           `mapstructure:"OUTBOX_RETENTION_DAYS"`
}

// LoadConfig reads outbox configuration from environment variables.
func LoadConfig() (Config, error) {
	var cfg Config

	viper.SetConfigName("app")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	viper.SetDefault("OUTBOX_POLL_INTERVAL", "1s")
	viper.SetDefault("OUTBOX_MAX_ATTEMPTS", 10)
	viper.SetDefault("OUTBOX_BATCH_SIZE", 100)
	viper.SetDefault("OUTBOX_DISPATCHER_ENABLED", true)
	viper.SetDefault("OUTBOX_BACKOFF_BASE", "1s")
	viper.SetDefault("OUTBOX_BACKOFF_MAX", "60s")
	viper.SetDefault("OUTBOX_RETENTION_DAYS", 14)

	if err := viper.ReadInConfig(); err == nil {
		_ = err
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}
