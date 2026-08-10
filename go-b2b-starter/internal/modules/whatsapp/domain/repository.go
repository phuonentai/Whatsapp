package domain

import (
	"context"
	"time"
)

type ConfigRepository interface {
	GetByPhoneNumberID(ctx context.Context, phoneNumberID string) (*WhatsAppConfig, error)
	GetByOrganizationID(ctx context.Context, orgID int32) (*WhatsAppConfig, error)
	GetByVerifyToken(ctx context.Context, verifyToken string) (*WhatsAppConfig, error)
	Create(ctx context.Context, config *WhatsAppConfig) (*WhatsAppConfig, error)
	Update(ctx context.Context, config *WhatsAppConfig) (*WhatsAppConfig, error)
}

type WebhookLogRepository interface {
	Insert(ctx context.Context, log *WebhookLog) (*WebhookLog, error)
	GetByID(ctx context.Context, id int32) (*WebhookLog, error)
	// InsertWithOutbox atomically persists the webhook log and its outbox
	// events in one transaction. Returns ErrDuplicateDelivery when the
	// delivery_key is already processed.
	InsertWithOutbox(ctx context.Context, log *WebhookLog, events []OutboxEventInput) (*WebhookLog, error)
	GetStatsByOrganization(ctx context.Context, orgID int32) (*WebhookLogStats, error)
}

type WebhookLogStats struct {
	Last24h        int              `json:"last_24h"`
	Last7d         int              `json:"last_7d"`
	Total          int              `json:"total"`
	ByStatus       map[string]int   `json:"by_status"`
	LastError      string           `json:"last_error,omitempty"`
	LastErrorAt    *time.Time       `json:"last_error_at,omitempty"`
}
