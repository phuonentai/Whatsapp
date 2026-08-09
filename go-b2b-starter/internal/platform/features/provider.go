package features

import "context"

// ModuleState is the runtime state of a module for an organization.
type ModuleState struct {
	Enabled  bool           `json:"enabled"`
	Features []string       `json:"features,omitempty"`
	Config   map[string]any `json:"config,omitempty"`
}

type Entitlement struct {
	Features      map[string]bool
	Quotas        map[string]int32
	Usage         map[string]int32
	IsReadOnly    bool
	IsGracePeriod bool
	PlanName      string
	// Modules holds the per-org module state keyed by module key.
	Modules map[string]ModuleState
}

type FeatureProvider interface {
	GetEntitlement(ctx context.Context, orgID int32) (*Entitlement, error)
}
