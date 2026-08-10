package domain

import (
	"context"
)

type ConfigRepository interface {
	GetByIGUserID(ctx context.Context, igUserID string) (*InstagramConfig, error)
	GetByOrganizationID(ctx context.Context, orgID int32) (*InstagramConfig, error)
	GetByVerifyToken(ctx context.Context, verifyToken string) (*InstagramConfig, error)
	Create(ctx context.Context, config *InstagramConfig) (*InstagramConfig, error)
	Update(ctx context.Context, config *InstagramConfig) (*InstagramConfig, error)
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
