package domain

import "errors"

var (
	ErrConfigNotFound          = errors.New("instagram config not found")
	ErrWebhookLogNotFound      = errors.New("instagram webhook log not found")
	ErrOrgRequired             = errors.New("organization ID is required")
	ErrIGUserIDRequired        = errors.New("IG user ID is required")
	ErrWebhookSecretRequired   = errors.New("webhook secret is required")
	ErrInvalidSignature        = errors.New("invalid webhook signature")
	ErrUnknownIGUser           = errors.New("unknown IG user ID")
	ErrWebhookVerificationFail = errors.New("webhook verification failed")
	ErrDuplicateDelivery       = errors.New("duplicate webhook delivery")
	ErrTokenRefreshFailed      = errors.New("instagram token refresh failed")
	ErrIGUserIDConflict        = errors.New("ig_user_id already in use by another organization")
)
