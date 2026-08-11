## ADDED Requirements

### Requirement: make test-e2e boots the mock invoicing provider

The `make test-e2e` bootstrap (`scripts/run_e2e.sh`) SHALL start the mock Siigo provider server before the backend boots, SHALL export the mock `SIIGO_BASE_URL`, `SIIGO_TOKEN_URL`, and `SIIGO_WEBHOOK_SECRET` configuration to the backend process, and SHALL terminate the mock server in the cleanup path.

#### Scenario: Mock provider started and cleaned up

- **WHEN** `make test-e2e` runs
- **THEN** the mock Siigo server SHALL be healthy before the backend starts
- **AND** the backend SHALL receive the mock provider configuration via environment
- **AND** on exit, the mock server SHALL be terminated with the other spawned processes
