## ADDED Requirements

### Requirement: Deal stage must belong to the deal's pipeline

The system SHALL enforce that a deal's `stage_id` always belongs to the deal's `pipeline_id`. The database SHALL constrain `crm.deals (organization_id, stage_id, pipeline_id)` against `crm.pipeline_stages (organization_id, id, pipeline_id)` via a composite foreign key, backed by a unique key `(organization_id, id, pipeline_id)` on `crm.pipeline_stages`.

#### Scenario: Deal with stage from another pipeline is rejected

- **WHEN** a deal is created or updated with a `stage_id` whose stage belongs to a different `pipeline_id` (same organization)
- **THEN** the database SHALL reject the statement with a foreign key violation

### Requirement: pipeline_id is derived from stage_id

The system SHALL keep `crm.deals.pipeline_id` synchronized with `crm.deals.stage_id` using a BEFORE trigger on `INSERT OR UPDATE OF stage_id`: the trigger SHALL set `pipeline_id` from the stage's `pipeline_id` (same organization), and SHALL raise an exception if the stage does not exist for the organization.

#### Scenario: Updating stage_id normalizes pipeline_id

- **WHEN** a deal's `stage_id` is updated to a stage of another pipeline
- **THEN** the trigger SHALL set the deal's `pipeline_id` to that stage's pipeline
- **AND** the update SHALL succeed (the composite FK then validates against the normalized value)

#### Scenario: Creating a deal with a matching stage and pipeline succeeds

- **WHEN** a deal is created with a `stage_id` and any `pipeline_id` value
- **THEN** the trigger SHALL normalize `pipeline_id` to the stage's pipeline before validation
- **AND** the insert SHALL succeed

#### Scenario: Stage deletion nulls deal stage and preserves pipeline

- **WHEN** a stage referenced by deals is deleted
- **THEN** `deals.stage_id` SHALL be set to NULL
- **AND** `deals.pipeline_id` SHALL remain unchanged
