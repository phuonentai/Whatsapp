## ADDED Requirements

### Requirement: Playbook catalog defines vertical business procedures

The system SHALL maintain a playbook registry (database-backed catalog) where each vertical playbook is defined by: a unique `key` (kebab-case, e.g., `comercio`), a display name, a business vertical, a description, an optional list of module dependencies (`requires_modules`), and a JSON payload defining the procedure seed data: a pipeline template (`nombre`, `etapas` with `nombre`/`orden`/`color`/`probabilidad`), a tag set, module configuration presets (`module_configs` keyed by module key), and message scripts (`guiones` with `id`/`titulo`/`mensaje`). The catalog SHALL be seeded from Go constants (single source) via migration.

#### Scenario: The five vertical playbooks ship complete

- **WHEN** the registry is seeded
- **THEN** playbooks SHALL exist with keys `comercio`, `restaurantes`, `citas`, `servicios-profesionales`, and `talleres`
- **AND** each SHALL have a non-empty name, non-empty description, a vertical, and a payload with at least one pipeline stage and at least one guion
- **AND** each pipeline stage SHALL have non-empty `nombre` and integer `orden`/`probabilidad` values
- **AND** each guion SHALL have non-empty `titulo` and `mensaje` text in Spanish

#### Scenario: Playbook payload validates against module registry

- **WHEN** a playbook declares a `module_configs` preset for a module (e.g., `tickets` with `{"sla_hours": 4}`)
- **THEN** the preset SHALL conform to that module's `config_schema` as registered in the module registry
- **AND** the system SHALL reject seeding data that violates the schema at startup or via tests

#### Scenario: Playbook dependencies reference only shipped modules

- **WHEN** a playbook declares `requires_modules`
- **THEN** every referenced key SHALL exist in the module registry
- **AND** playbooks SHALL NOT grant features or gate entitlements

### Requirement: Playbook application is one-way and idempotent

The system SHALL provide an apply operation (`POST /api/playbooks/{key}/apply`) that seeds, in a single transaction: the vertical pipeline (created as `es_predeterminado` only when the organization has no pipelines), missing tags by name, module config presets for enabled modules (validated against each module's `config_schema` and upserted into per-org module state), and the organization's playbook state (`organization_playbooks` with `applied_at`). Re-applying SHALL add only missing seed data and SHALL NOT delete or overwrite organization-created data.

#### Scenario: First apply seeds the organization

- **WHEN** an organization with no pipelines and no playbook applies the `comercio` playbook
- **THEN** a pipeline named per the playbook SHALL be created and marked predeterminado
- **AND** its stages SHALL match the playbook's `etapas` in order
- **AND** the playbook's tags SHALL be created
- **AND** module config presets SHALL be persisted for the organization's enabled modules
- **AND** the organization's playbook state SHALL record the applied playbook key and timestamp

#### Scenario: Re-apply only adds missing seed data

- **WHEN** the same organization re-applies the playbook after manually editing the pipeline or tags
- **THEN** the system SHALL NOT delete or modify the organization's edited pipeline or tags
- **AND** SHALL add any seed rows still missing (e.g., tags the organization deleted)
- **AND** SHALL NOT duplicate existing rows

#### Scenario: Apply skips pipeline when the organization already has pipelines

- **WHEN** an organization that already has pipelines applies a playbook
- **THEN** the playbook SHALL be applied without creating or altering any pipeline
- **AND** the apply SHALL NOT fail

#### Scenario: Module config preset for a disabled module is skipped

- **WHEN** a playbook contains a `module_configs` preset for a module the organization has not enabled
- **THEN** the preset SHALL be skipped without error
- **AND** the organization's module state SHALL NOT change for that module

#### Scenario: Invalid module config preset is rejected

- **WHEN** a playbook preset violates the module's `config_schema` (e.g., `sla_hours` is not a number)
- **THEN** the apply SHALL return HTTP 400 with a JSON error body
- **AND** SHALL NOT persist any partial seed data

### Requirement: Playbook reset removes only seeded data

The system SHALL provide a reset operation (`POST /api/playbooks/{key}/reset`) that removes playbook-seeded data: the organization's playbook state row, seeded tags that are not referenced by any entity, the seeded pipeline only when it has no deals, and seeded module config keys only when their stored value matches the playbook preset. Reset SHALL NOT touch organization-created data.

#### Scenario: Reset removes seeded rows

- **WHEN** an organization resets a playbook whose seeded pipeline has no deals and whose seeded tags are unreferenced
- **THEN** the pipeline, the unreferenced tags, the seeded config presets matching the playbook, and the playbook state row SHALL be removed

#### Scenario: Reset preserves protected seeded rows

- **WHEN** an organization resets a playbook whose seeded pipeline contains deals or whose seeded tags are referenced by entities
- **THEN** the pipeline SHALL NOT be deleted
- **AND** the referenced tags SHALL NOT be deleted
- **AND** the reset SHALL NOT fail

### Requirement: Playbook API is gated and exposes catalog with state and guiones

The system SHALL expose `GET /api/playbooks` returning the non-internal playbook catalog, each organization's applied playbook state, and the guiones of applied playbooks. Apply and reset SHALL be gated by active-subscription entitlement and existing RBAC scopes, mirroring CRM route conventions. No new Stytch RBAC policy changes are introduced.

#### Scenario: Catalog lists playbooks with applied state

- **WHEN** an authenticated organization with an active subscription requests `GET /api/playbooks`
- **THEN** the response SHALL include all five vertical playbooks
- **AND** SHALL include the organization's applied playbook key(s) and timestamps
- **AND** SHALL include the guiones of applied playbooks

#### Scenario: Apply requires an active subscription

- **WHEN** a request to apply a playbook is made by an organization without an active subscription
- **THEN** the system SHALL return the same entitlement-gating error as other subscription-gated routes
- **AND** SHALL NOT apply the playbook

#### Scenario: Apply for an unknown playbook returns 404

- **WHEN** a request to apply or reset a playbook key not present in the registry is made
- **THEN** the system SHALL return HTTP 404 with a JSON error body

### Requirement: Guiones surface as quick replies executed by agents

The system SHALL surface the applied playbook's guiones as quick-reply suggestions in the WhatsApp inbox UI. Selecting a guion SHALL fill the composer with its message text for a human agent to send via the existing conversation send endpoint; the platform SHALL NOT auto-send guiones.

#### Scenario: Inbox shows quick replies for applied playbook

- **WHEN** an agent opens a conversation in an organization that has applied a playbook with guiones
- **THEN** the inbox SHALL render a quick-reply chip row listing the playbook's guiones
- **AND** selecting a chip SHALL prefill the message composer with the guion's `mensaje`

#### Scenario: No playbook means no quick replies

- **WHEN** an agent opens a conversation in an organization that has not applied any playbook
- **THEN** the inbox SHALL NOT render the quick-reply chip row
