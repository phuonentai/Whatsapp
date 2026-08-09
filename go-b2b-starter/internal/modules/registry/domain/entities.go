package domain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrModuleNotFound      = errors.New("module not found")
	ErrModuleDisabled      = errors.New("module disabled")
	ErrInvalidModuleConfig = errors.New("invalid module config")
)

// Module is a sellable capability in the product catalog.
type Module struct {
	ID              int32
	Key             string
	Name            string
	Description     string
	GrantedFeatures []string
	Requires        []string
	ConfigSchema    json.RawMessage
	IsInternal      bool
	IsActive        bool
}

// HasFeature reports whether the module grants the given feature key.
func (m *Module) HasFeature(feature string) bool {
	for _, f := range m.GrantedFeatures {
		if f == feature {
			return true
		}
	}
	return false
}

// OrganizationModule is the per-org state and config of a module.
type OrganizationModule struct {
	OrganizationID int32
	ModuleKey      string
	Config         map[string]any
	EnabledAt      string
}

// ModuleRepository persists the module catalog.
type ModuleRepository interface {
	// ListActive returns all active modules (including internal, for the vendor).
	ListActive(ctx context.Context) ([]*Module, error)
	// GetByKey returns a module by key; ErrModuleNotFound if absent.
	GetByKey(ctx context.Context, key string) (*Module, error)
}

// OrganizationModuleRepository persists per-org module state and config.
type OrganizationModuleRepository interface {
	// ListByOrg returns all modules enabled for the org (state rows).
	ListByOrg(ctx context.Context, orgID int32) ([]*OrganizationModule, error)
	// GetByKey returns the org's state for one module; nil if not enabled.
	GetByKey(ctx context.Context, orgID int32, moduleKey string) (*OrganizationModule, error)
	// UpsertConfig saves (or creates) the org's module state with config.
	UpsertConfig(ctx context.Context, orgID int32, moduleKey string, config map[string]any) (*OrganizationModule, error)
	// Delete removes the module from the org.
	Delete(ctx context.Context, orgID int32, moduleKey string) error
}

// ModuleConfigSchema describes the supported per-org config keys for the tickets module.
// Validated by app/services; kept here as the domain contract.
var ModuleConfigSchema = struct {
	SLAHoursKey   string
	PrioritiesKey string
	TagsKey       string
}{
	SLAHoursKey:   "sla_hours",
	PrioritiesKey: "priorities",
	TagsKey:       "tags",
}

func (m *Module) String() string {
	return fmt.Sprintf("Module{key=%s, internal=%v, active=%v}", m.Key, m.IsInternal, m.IsActive)
}
