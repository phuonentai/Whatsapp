package domain

import "errors"

var (
	ErrConfigNotFound          = errors.New("whatsapp config not found")
	ErrWebhookLogNotFound      = errors.New("webhook log not found")
	ErrOrgRequired             = errors.New("organization ID is required")
	ErrPhoneNumberIDRequired   = errors.New("phone number ID is required")
	ErrWebhookSecretRequired   = errors.New("webhook secret is required")
	ErrInvalidSignature        = errors.New("invalid webhook signature")
	ErrUnknownPhoneNumber      = errors.New("unknown phone number ID")
	ErrWebhookVerificationFail = errors.New("webhook verification failed")
)
