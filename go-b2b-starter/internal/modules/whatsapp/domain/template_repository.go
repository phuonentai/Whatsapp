package domain

import "context"

// TemplateRepository is the org-scoped persistence boundary for the template
// registry. Every method filters by organization_id; the caller resolves the
// org from the authenticated request context (Stytch B2B SSOT).
type TemplateRepository interface {
	Create(ctx context.Context, template *Template) (*Template, error)
	GetByID(ctx context.Context, orgID int32, id int64) (*Template, error)
	GetByOrgNameLanguage(ctx context.Context, orgID int32, name, language string) (*Template, error)
	GetByMetaTemplateID(ctx context.Context, orgID int32, metaTemplateID string) (*Template, error)
	ListByOrg(ctx context.Context, orgID int32) ([]*Template, error)
	Update(ctx context.Context, template *Template) (*Template, error)
	// UpdateStatus applies a status change with a transaction-isolated state
	// check: it returns (nil, nil) when the stored status already equals the
	// target (re-delivered webhook / redundant refresh is a no-op).
	UpdateStatus(ctx context.Context, orgID int32, id int64, status TemplateStatus, metaTemplateID, rejectionReason *string) (*Template, error)
	Delete(ctx context.Context, orgID int32, id int64) error
	CountByOrg(ctx context.Context, orgID int32) (int64, error)
}
