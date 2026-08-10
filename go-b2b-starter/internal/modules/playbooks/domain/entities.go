package domain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrPlaybookNotFound          = errors.New("playbook not found")
	ErrInvalidPlaybookPayload    = errors.New("invalid playbook payload")
	ErrPlaybookRequiresModules   = errors.New("playbook requires disabled modules")
	ErrInvalidModuleConfigPreset = errors.New("invalid module config preset")
)

// Playbook is a vertical business procedure package in the catalog.
type Playbook struct {
	ID              int32
	Key             string
	Name            string
	Vertical        string
	Description     string
	RequiresModules []string
	Payload         json.RawMessage
	IsActive        bool
}

// PlaybookPayload is the machine-readable procedure seed data of a playbook.
type PlaybookPayload struct {
	Pipeline      PipelineTemplate          `json:"pipeline"`
	Tags          []string                  `json:"tags"`
	ModuleConfigs map[string]map[string]any `json:"module_configs"`
	Guiones       []Guion                   `json:"guiones"`
}

// PipelineTemplate defines the vertical pipeline to seed.
type PipelineTemplate struct {
	Nombre string          `json:"nombre"`
	Etapas []EtapaTemplate `json:"etapas"`
}

// EtapaTemplate defines one pipeline stage to seed.
type EtapaTemplate struct {
	Nombre       string `json:"nombre"`
	Orden        int32  `json:"orden"`
	Color        string `json:"color"`
	Probabilidad int32  `json:"probabilidad"`
}

// Guion is a WhatsApp message script surfaced as a quick reply. A guion is
// either a single-shot message (Mensaje) or a scripted sequence (Pasos).
type Guion struct {
	ID      string      `json:"id"`
	Titulo  string      `json:"titulo"`
	Mensaje string      `json:"mensaje,omitempty"`
	Pasos   []GuionPaso `json:"pasos,omitempty"`
}

// GuionPaso is one ordered step of a scripted sequence.
type GuionPaso struct {
	ID      string `json:"id"`
	Titulo  string `json:"titulo"`
	Mensaje string `json:"mensaje"`
}

// OrganizationPlaybook is the per-org record of an applied playbook.
type OrganizationPlaybook struct {
	OrganizationID   int32
	PlaybookKey      string
	SeededPipelineID *int32
	AppliedAt        string
}

// PlaybookRepository persists the playbook catalog.
type PlaybookRepository interface {
	// ListActive returns all active playbooks.
	ListActive(ctx context.Context) ([]*Playbook, error)
	// GetByKey returns a playbook by key; ErrPlaybookNotFound if absent.
	GetByKey(ctx context.Context, key string) (*Playbook, error)
}

// OrganizationPlaybookRepository persists per-org playbook state.
type OrganizationPlaybookRepository interface {
	// ListByOrg returns all playbooks applied by the org.
	ListByOrg(ctx context.Context, orgID int32) ([]*OrganizationPlaybook, error)
	// GetByOrgKey returns the org's state for one playbook; nil if not applied.
	GetByOrgKey(ctx context.Context, orgID int32, key string) (*OrganizationPlaybook, error)
	// Upsert records (or refreshes) the org's applied playbook state.
	Upsert(ctx context.Context, orgID int32, key string, seededPipelineID *int32) (*OrganizationPlaybook, error)
	// Delete removes the org's playbook state.
	Delete(ctx context.Context, orgID int32, key string) error
}

// PlaybookApplyRepository seeds and reverts playbook data atomically.
type PlaybookApplyRepository interface {
	// Apply seeds the playbook's pipeline, tags, and module config presets in
	// one transaction (one-way and additive). Returns the org playbook state.
	Apply(ctx context.Context, orgID int32, pb *Playbook, payload *PlaybookPayload) (*OrganizationPlaybook, error)
	// Reset removes playbook-seeded data only (config presets still matching,
	// unreferenced tags, seeded pipeline without deals). Never touches
	// organization-created data.
	Reset(ctx context.Context, orgID int32, pb *Playbook, payload *PlaybookPayload) error
}

func (p *Playbook) String() string {
	return fmt.Sprintf("Playbook{key=%s, vertical=%s, active=%v}", p.Key, p.Vertical, p.IsActive)
}
