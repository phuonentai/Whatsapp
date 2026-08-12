## MODIFIED Requirements

### Requirement: Test database seeded with test organizations

The system SHALL seed the test database via the `cmd/seed-e2e` command, invoked by the canonical e2e bootstrap and by CI after migrations are applied. The Playwright `global-setup.ts` SHALL validate that the seeded organizations exist; it SHALL NOT create them.

#### Scenario: Bootstrap seeds test orgs before the suite runs
- **WHEN** `make test-e2e` (via `scripts/run_e2e.sh`) or a CI e2e job runs
- **THEN** database migrations are applied to `saas_db_test`
- **AND** `cmd/seed-e2e` SHALL create `test-org-free` (Free plan), `test-org-pro` (Pro plan), `test-org-enterprise` (Enterprise plan), and `test-org-rbac` (Pro plan)
- **AND** `seed-e2e` SHALL create an admin account for each org and manager and member accounts for `test-org-rbac`
- **AND** the suite boots only after seeding completes

#### Scenario: Global setup validates preconditions without seeding
- **WHEN** `global-setup.ts` runs
- **THEN** it SHALL verify the expected test orgs exist (or log instructions to run `make test-e2e`)
- **AND** it SHALL NOT create or modify organizations, accounts, subscriptions, or quotas
