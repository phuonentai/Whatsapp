package graphapi

import "os"

const (
	EnvAppID         = "INSTAGRAM_APP_ID"
	EnvAppSecret     = "INSTAGRAM_APP_SECRET"
	EnvWebhookVerify = "INSTAGRAM_WEBHOOK_VERIFY_TOKEN"
	EnvAPIBase       = "INSTAGRAM_API_BASE"
	EnvAPIVersion    = "INSTAGRAM_API_VERSION"

	defaultAPIBase    = "https://graph.facebook.com"
	defaultAPIVersion = "v21.0"
)

// FromEnv builds the Instagram Graph API client config. Unlike the WhatsApp
// signup path, Instagram is an optional feature: missing vars fall back to
// defaults and the webhook verify token may be empty (per-config tokens then
// apply).
func FromEnv() ClientConfig {
	return ClientConfig{
		AppID:      os.Getenv(EnvAppID),
		AppSecret:  os.Getenv(EnvAppSecret),
		APIBase:    envOr(EnvAPIBase, defaultAPIBase),
		APIVersion: envOr(EnvAPIVersion, defaultAPIVersion),
	}
}

// WebhookVerifyToken returns the platform-level hub verify token for the
// Instagram webhook handshake, or "" when unset.
func WebhookVerifyToken() string {
	return os.Getenv(EnvWebhookVerify)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
