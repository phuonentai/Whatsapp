## MODIFIED Requirements

### Requirement: CI runs spec validation on every push and pull request

The CI pipeline SHALL run `openspec validate --specs` AND `openspec validate --specs --strict` in a dedicated job on every push and pull request. The job SHALL fail the build when any living spec is invalid (e.g., missing `## Purpose` or `## Requirements` sections, malformed requirement or scenario blocks) or when any spec raises a warning under strict mode (e.g., `## Purpose` shorter than 50 characters). The non-strict run SHALL catch structural errors; the strict run SHALL catch framing warnings; a failure of either run SHALL fail the job.

#### Scenario: Specs tree is valid

- **WHEN** the CI `spec-validation` job runs
- **AND** all living specs pass `openspec validate --specs`
- **AND** all living specs pass `openspec validate --specs --strict`
- **THEN** the job SHALL exit 0 and the build continues

#### Scenario: Spec tree regresses structurally

- **WHEN** the CI `spec-validation` job runs
- **AND** at least one living spec fails `openspec validate --specs`
- **THEN** the job SHALL exit non-zero
- **AND** the check SHALL report which spec(s) failed validation

#### Scenario: Spec Purpose regresses below the length floor

- **WHEN** the CI `spec-validation` job runs
- **AND** all specs pass the non-strict run
- **AND** at least one living spec fails `openspec validate --specs --strict` (e.g., `## Purpose` under 50 characters)
- **THEN** the job SHALL exit non-zero
- **AND** the check SHALL report which spec(s) failed strict validation
