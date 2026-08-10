## ADDED Requirements

### Requirement: Guiones can be scripted sequences of ordered steps

A guion in a playbook payload SHALL be either a single-shot message (non-empty `mensaje`) or a scripted sequence (non-empty `pasos` array of 2+ ordered steps, each step with non-empty `id`, `titulo`, and `mensaje` in Spanish). A guion with a `pasos` array SHALL NOT require `mensaje` and SHALL NOT be both a sequence and a single-shot message. Payload validation SHALL reject incomplete sequences (step missing `id`, `titulo`, or `mensaje`, or fewer than 2 steps) with the existing invalid-payload error path.

#### Scenario: Single-shot guion remains valid

- **WHEN** a playbook payload contains a guion with a non-empty `mensaje` and no `pasos`
- **THEN** the payload SHALL validate and the guion SHALL behave as today (fills the composer on click)

#### Scenario: Sequence guion validates with complete steps

- **WHEN** a playbook payload contains a guion with a `pasos` array of 2+ steps, each with non-empty `id`, `titulo`, and `mensaje`
- **THEN** the payload SHALL validate and the guion SHALL be exposed as a scripted sequence

#### Scenario: Incomplete sequence is rejected

- **WHEN** a playbook payload contains a guion whose `pasos` has fewer than 2 steps or any step missing `id`, `titulo`, or `mensaje`
- **THEN** the payload SHALL be rejected as invalid and SHALL NOT be persisted

### Requirement: Catalog and migration seed ship sequence guiones per vertical

The playbook catalog (`catalog.go`) and the mirrored migration seed SHALL include at least one scripted sequence guion per vertical, ordered and complete. The startup `CatalogValidated` check SHALL keep the Go catalog and the seeded DB payloads in sync, including `pasos`.

#### Scenario: All five verticals expose a sequence guion

- **WHEN** the playbook catalog is loaded for `comercio`, `restaurantes`, `citas`, `servicios-profesionales`, or `talleres`
- **THEN** each SHALL contain at least one guion with a valid `pasos` array

#### Scenario: Seeded DB payloads match the catalog

- **WHEN** the startup validation compares each vertical's seeded `playbooks.payload` against the Go catalog
- **THEN** the guiones (including `pasos`) SHALL match and validation SHALL pass

### Requirement: Catalog API exposes sequence steps

`GET /api/playbooks` SHALL include, for each applied playbook, the guiones with their `pasos` arrays when present, alongside the existing `id`/`titulo`/`mensaje` fields.

#### Scenario: Applied playbook returns sequence steps

- **WHEN** an organization has applied a playbook whose guiones include a sequence
- **THEN** the catalog response SHALL include the sequence guion with its ordered `pasos` (id, titulo, mensaje)
- **AND** single-shot guiones SHALL be returned as today without a `pasos` field

#### Scenario: Non-applied playbook does not expose guiones

- **WHEN** a playbook is not applied for the organization
- **THEN** its guiones (including any `pasos`) SHALL NOT appear in the response
