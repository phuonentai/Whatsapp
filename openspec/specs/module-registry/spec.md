# Spec: module-registry

## Purpose

Module registry and per-org module configuration behaviour, including the shared validation path used by manual edits and playbook presets.

## Requirements

### Requirement: Module configuration presets from playbooks use the same validation path

The system SHALL accept per-org module configuration written through playbook application (vertical playbook `module_configs` presets) through the same validation and persistence path as manual configuration edits: presets SHALL be validated against the module's `config_schema` before persistence, SHALL be rejected with HTTP 400 (with no partial persistence) when invalid, and SHALL be skipped without error when the target module is not enabled for the organization.

#### Scenario: Playbook preset is validated and persisted

- **WHEN** a playbook preset for the `tickets` module (`{"sla_hours": 4, "priorities": ["low","normal","high"]}`) is applied for an organization with the `tickets` module enabled
- **THEN** the preset SHALL be validated against the `tickets` config schema
- **AND** SHALL be persisted into the organization's module state for `tickets`
- **AND** SHALL be returned by the entitlement API

#### Scenario: Invalid playbook preset is rejected with no partial persistence

- **WHEN** a playbook preset violates the module's `config_schema` (e.g., `sla_hours` is not a number)
- **THEN** the system SHALL return HTTP 400 with a JSON error body
- **AND** SHALL NOT persist the preset or any other partial seed data from the apply

#### Scenario: Playbook preset for a disabled module is skipped

- **WHEN** a playbook preset targets a module that is not enabled for the organization
- **THEN** the preset SHALL be skipped without error
- **AND** the organization's module state SHALL NOT change for that module
