package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_CommissionRateDefaultsToZero(t *testing.T) {
	t.Setenv("PAYMENTS_COMMISSION_RATE", "")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, 0.0, cfg.CommissionRate)
}

func TestLoadConfig_ReadsCommissionRate(t *testing.T) {
	t.Setenv("PAYMENTS_COMMISSION_RATE", "0.025")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, 0.025, cfg.CommissionRate)
}

func TestLoadConfig_RejectsOutOfRangeRate(t *testing.T) {
	t.Setenv("PAYMENTS_COMMISSION_RATE", "1.5")

	_, err := LoadConfig()
	require.Error(t, err)
}

func TestLoadConfig_RejectsNegativeRate(t *testing.T) {
	t.Setenv("PAYMENTS_COMMISSION_RATE", "-0.1")

	_, err := LoadConfig()
	require.Error(t, err)
}
