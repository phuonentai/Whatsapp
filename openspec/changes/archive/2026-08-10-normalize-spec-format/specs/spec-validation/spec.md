## ADDED Requirements

### Requirement: CI runs spec validation on every push and pull request

The CI pipeline SHALL run `openspec validate --specs` in a dedicated job on every push and pull request. The job SHALL fail the build when any living spec is invalid (e.g., missing `## Purpose` or `## Requirements` sections, malformed requirement or scenario blocks).

#### Scenario: Specs tree is valid

- **WHEN** the CI `spec-validation` job runs
- **AND** all living specs pass `openspec validate --specs`
- **THEN** the job SHALL exit 0 and the build continues

#### Scenario: Spec tree regresses

- **WHEN** the CI `spec-validation` job runs
- **AND** at least one living spec fails `openspec validate --specs`
- **THEN** the job SHALL exit non-zero
- **AND** the check SHALL report which spec(s) failed validation
