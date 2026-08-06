package features

import "context"

type Entitlement struct {
	Features      map[string]bool
	Quotas        map[string]int32
	Usage         map[string]int32
	IsReadOnly    bool
	IsGracePeriod bool
	PlanName      string
}

type FeatureProvider interface {
	GetEntitlement(ctx context.Context, orgID int32) (*Entitlement, error)
}
