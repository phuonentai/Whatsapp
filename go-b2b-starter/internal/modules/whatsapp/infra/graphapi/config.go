package graphapi

import (
	"fmt"
	"os"
)

const (
	EnvAppID          = "WHATSAPP_APP_ID"
	EnvAppSecret      = "WHATSAPP_APP_SECRET"
	EnvSignupConfigID = "WHATSAPP_SIGNUP_CONFIG_ID"
	EnvRedirectURI    = "WHATSAPP_REDIRECT_URI"
	EnvCallbackURL    = "WHATSAPP_WEBHOOK_CALLBACK_URL"
	EnvAPIBase        = "WHATSAPP_API_BASE"
	EnvAPIVersion     = "WHATSAPP_API_VERSION"

	defaultAPIBase    = "https://graph.facebook.com"
	defaultAPIVersion = "v21.0"
	defaultCallback   = "http://localhost:8080/api/v1/webhooks/whatsapp"
)

// FromEnv builds the Graph API client config and the browser bootstrap config
// from validated environment invariants. Required vars missing from the
// environment fail loudly so misconfiguration surfaces at startup.
func FromEnv() (ClientConfig, MetaConfig, error) {
	appID := os.Getenv(EnvAppID)
	appSecret := os.Getenv(EnvAppSecret)
	configID := os.Getenv(EnvSignupConfigID)

	if appID == "" {
		return ClientConfig{}, MetaConfig{}, fmt.Errorf("required env var %s is not set", EnvAppID)
	}
	if appSecret == "" {
		return ClientConfig{}, MetaConfig{}, fmt.Errorf("required env var %s is not set", EnvAppSecret)
	}
	if configID == "" {
		return ClientConfig{}, MetaConfig{}, fmt.Errorf("required env var %s is not set", EnvSignupConfigID)
	}

	apiBase := os.Getenv(EnvAPIBase)
	if apiBase == "" {
		apiBase = defaultAPIBase
	}
	apiVersion := os.Getenv(EnvAPIVersion)
	if apiVersion == "" {
		apiVersion = defaultAPIVersion
	}

	return ClientConfig{
			AppID:      appID,
			AppSecret:  appSecret,
			APIBase:    apiBase,
			APIVersion: apiVersion,
		}, MetaConfig{
			AppID:       appID,
			ConfigID:    configID,
			RedirectURI: os.Getenv(EnvRedirectURI),
		}, nil
}

// CallbackURL returns the public webhook callback URL used for app subscriptions.
func CallbackURL() string {
	if v := os.Getenv(EnvCallbackURL); v != "" {
		return v
	}
	return defaultCallback
}
