## ADDED Requirements

### Requirement: Module catalog is the source of truth for sellable modules

The system SHALL maintain a module registry (database-backed catalog) where each sellable module is defined by: a unique `key` (kebab-case, e.g., `tickets`), a display name, a description, the set of feature keys it grants, the list of module dependencies (`requires`), an optional JSON config schema, and an `is_internal` flag marking vendor-only modules (e.g., org #0 ops tooling).

#### Scenario: Module definition is complete

- **WHEN** the registry is seeded with the `tickets` module
- **THEN** the module SHALL have key `tickets`, a non-empty display name, the feature key `tickets_module` in its granted features, an empty dependency list, and `is_internal` equal to false

#### Scenario: Internal module is flagged vendor-only

- **WHEN** the registry contains an internal module (e.g., `ops-internal`)
- **THEN** `is_internal` SHALL be true for that module
- **AND** the module SHALL be excluded from tenant-facing catalog listings

#### Scenario: Module dependency is enforced

- **WHEN** a module declares a dependency on another module (e.g., `tickets` requires `inbox`)
- **THEN** the dependent module's features SHALL NOT be considered enabled for an organization unless the dependency module is also enabled for that organization

### Requirement: Organizations hold per-org module state and configuration

The system SHALL persist, per organization: the set of enabled modules and a JSONB configuration object per module. Configuration SHALL be validated against the module's config schema when saved.

#### Scenario: Module config is saved and validated

- **WHEN** an organization with the `tickets` module enabled saves a config `{"sla_hours": 24, "priorities": ["low","normal","high"], "tags": ["billing","bug"]}`
- **THEN** the config SHALL be persisted and returned by the entitlement API

#### Scenario: Invalid module config is rejected

- **WHEN** an organization saves a config that violates the module's config schema (e.g., `sla_hours` is not a number)
- **THEN** the system SHALL return HTTP 400 with a JSON error body
- **AND** SHALL NOT persist the config

### Requirement: Entitlement derivation unions plan features with purchased module features

`FeatureProvider.GetEntitlement` SHALL derive the organization's feature set as the union of base-plan features (existing `crm_features` semantics) and features granted by the organization's enabled modules. Module feature keys SHALL be namespaced (e.g., `tickets_module`) to avoid collision with plan features. Quotas and usage SHALL remain unchanged in semantics.

#### Scenario: Module adds features on top of plan

- **WHEN** a Pro-plan organization with no modules requests entitlement
- **THEN** `crm_deals` SHALL be true
- **AND** `tickets_module` SHALL be false

- **WHEN** the same organization subsequently has the `tickets` module enabled
- **THEN** `crm_deals` SHALL still be true
- **AND** `tickets_module` SHALL be true

#### Scenario: No subscription means no module features

- **WHEN** an organization has no active subscription
- **THEN** all features SHALL be false, including module features
- **AND** the WhatsApp bridge SHALL still function (not feature-gated)

#### Scenario: Unknown module key is ignored

- **WHEN** subscription metadata contains a module key not present in the registry
- **THEN** the unknown key SHALL be ignored
- **AND** the organization's module state SHALL NOT change

### Requirement: Module-gating middleware rejects disabled modules

The system SHALL provide module-gating middleware (e.g., `modules.Require("tickets")`) that returns HTTP 403 when the module is not enabled for the requesting organization, mirroring `features.Require` semantics and ordering (feature/module gate runs before permission checks).

#### Scenario: Enabled module allows access

- **WHEN** a request passes through `modules.Require("tickets")` for an organization with `tickets` enabled
- **THEN** the request SHALL proceed to the handler

#### Scenario: Disabled module returns 403

- **WHEN** a request passes through `modules.Require("tickets")` for an organization without the `tickets` module
- **THEN** the system SHALL return HTTP 403 with a JSON body `{"error": "module_disabled", "module": "tickets"}`
- **AND** SHALL abort the request

### Requirement: Entitlement endpoint exposes module state and configuration

The entitlement endpoint SHALL include, alongside features/quotas/usage: the list of enabled modules, each module's granted feature keys, and each enabled module's configuration. Module state SHALL be cached per request alongside features.

#### Scenario: Entitlement response includes modules

- **WHEN** an organization with the `tickets` module enabled requests the entitlement endpoint
- **THEN** the response SHALL include `tickets` under enabled modules
- **AND** the response SHALL include the `tickets` module configuration object

#### Scenario: Internal modules are hidden from tenant entitlement responses

- **WHEN** a non-vendor organization requests the entitlement endpoint
- **THEN** internal modules SHALL NOT appear in the response

### Requirement: Module enablement follows the billing pipeline

The system SHALL derive enabled modules from subscription metadata (namespaced keys, e.g., `module_tickets`), consistent with the existing `crm_features` pattern, so that module purchases surfaced by Polar/MercadoPago webhooks enable modules without per-sale code changes.

#### Scenario: Purchase appears in subscription metadata

- **WHEN** subscription metadata contains `module_tickets` for an organization
- **THEN** the `tickets` module SHALL be enabled for that organization
- **AND** the `tickets_module` feature SHALL be true in entitlement

#### Scenario: Module removed from metadata disables module

- **WHEN** subscription metadata no longer contains `module_tickets` for an organization
- **THEN** the `tickets` module SHALL be disabled for that organization
- **AND** the `tickets_module` feature SHALL be false in entitlement
