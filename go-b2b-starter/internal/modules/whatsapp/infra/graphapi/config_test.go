package graphapi

import (
	"testing"
)

func TestFromEnv_Success(t *testing.T) {
	t.Setenv(EnvAppID, "app-1")
	t.Setenv(EnvAppSecret, "secret-1")
	t.Setenv(EnvSignupConfigID, "cfg-1")

	cfg, meta, err := FromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppID != "app-1" || cfg.AppSecret != "secret-1" {
		t.Fatalf("unexpected client config: %+v", cfg)
	}
	if meta.ConfigID != "cfg-1" {
		t.Fatalf("unexpected meta config: %+v", meta)
	}
	if cfg.APIBase != defaultAPIBase || cfg.APIVersion != defaultAPIVersion {
		t.Fatalf("expected defaults, got base=%s version=%s", cfg.APIBase, cfg.APIVersion)
	}
}

func TestFromEnv_MissingAppIDFails(t *testing.T) {
	t.Setenv(EnvAppID, "")
	t.Setenv(EnvAppSecret, "secret-1")
	t.Setenv(EnvSignupConfigID, "cfg-1")

	_, _, err := FromEnv()
	if err == nil {
		t.Fatal("expected error when WHATSAPP_APP_ID is missing")
	}
}

func TestFromEnv_MissingAppSecretFails(t *testing.T) {
	t.Setenv(EnvAppID, "app-1")
	t.Setenv(EnvAppSecret, "")
	t.Setenv(EnvSignupConfigID, "cfg-1")

	_, _, err := FromEnv()
	if err == nil {
		t.Fatal("expected error when WHATSAPP_APP_SECRET is missing")
	}
}

func TestFromEnv_MissingConfigIDFails(t *testing.T) {
	t.Setenv(EnvAppID, "app-1")
	t.Setenv(EnvAppSecret, "secret-1")
	t.Setenv(EnvSignupConfigID, "")

	_, _, err := FromEnv()
	if err == nil {
		t.Fatal("expected error when WHATSAPP_SIGNUP_CONFIG_ID is missing")
	}
}
