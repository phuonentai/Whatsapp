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

	ErrSignupNotFound         = errors.New("signup flow not found")
	ErrSignupCodeRequired     = errors.New("authorization code is required")
	ErrSignupInProgress       = errors.New("signup already in progress")
	ErrSignupAlreadyConnected = errors.New("signup already connected")
)
