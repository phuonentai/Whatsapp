package domain

import "errors"

var (
	// ErrTemplateNotFound: the template does not exist for the organization.
	ErrTemplateNotFound = errors.New("template_not_found")
	// ErrTemplateNameConflict: (organization_id, name, language) already exists.
	ErrTemplateNameConflict = errors.New("template_name_conflict")
	// ErrTemplateNotDraft: draft-only operations (update/delete) rejected.
	ErrTemplateNotDraft = errors.New("template_not_draft")
	// ErrTemplateNotApproved: send requires local status 'approved'.
	ErrTemplateNotApproved = errors.New("template_not_approved")
	// ErrTemplateParamCountMismatch: len(params) != param_count at send time.
	ErrTemplateParamCountMismatch = errors.New("template_param_count_mismatch")
	// ErrTemplateInvalidTransition: forbidden local state machine transition.
	ErrTemplateInvalidTransition = errors.New("template_invalid_transition")
	// ErrTemplateNotFoundAtMeta: manual refresh found no template at Meta.
	ErrTemplateNotFoundAtMeta = errors.New("template_not_found_at_meta")
	// ErrTemplateStatusSyncConflict: local/Meta status cannot be reconciled.
	ErrTemplateStatusSyncConflict = errors.New("template_status_sync_conflict")
)
