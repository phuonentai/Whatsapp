## Purpose

Define the E2E test infrastructure for the CRM suite: a Playwright project in `next_b2b_starter/e2e/`, a mock authentication middleware gated by `AUTH_MOCK_ENABLED`, a seeded test database, and shared fixtures/page objects.
## Requirements
### Requirement: Playwright project configured in next_b2b_starter

The system SHALL have a Playwright test project at `next_b2b_starter/e2e/` with TypeScript configuration.

#### Scenario: Playwright config exists
- **WHEN** `playwright.config.ts` is loaded
- **THEN** it SHALL specify test directory `./e2e/specs`, Chromium-only projects, and base URL from environment

### Requirement: Mock auth middleware for test mode

The system SHALL provide a mock authentication middleware activated by `AUTH_MOCK_ENABLED=true` that reads `X-Test-Org-ID` header to create a mock session.

#### Scenario: Mock header creates valid session
- **WHEN** a request includes `X-Test-Org-ID: test-org-pro`
- **THEN** the middleware SHALL create a session with the matching org ID and account from the seeded DB
- **AND** SHALL bypass all Stytch validation

#### Scenario: Missing mock header falls through to real auth
- **WHEN** `AUTH_MOCK_ENABLED=true` but no `X-Test-Org-ID` header is present
- **THEN** the middleware SHALL return 401

#### Scenario: Mock auth disabled in production
- **WHEN** `AUTH_MOCK_ENABLED` is `false` or unset
- **THEN** the middleware SHALL pass through to normal Stytch JWT validation

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

### Requirement: Shared test fixtures and page objects

The system SHALL provide reusable page object classes for each CRM entity and shared fixtures for authentication.

#### Scenario: Page objects expose entity-specific interactions
- **WHEN** a test imports a page object (e.g., `ContactsPage`)
- **THEN** it SHALL provide methods for create, read, update, delete, search, and filter operations for that entity

### Requirement: Playwright suite runs with parallel workers

The system SHALL configure the Playwright E2E suite to execute tests in parallel across multiple workers, with worker count derived from available CPU cores locally and an explicit bounded count in CI.

#### Scenario: Parallel execution enabled
- **WHEN** `playwright.config.ts` is loaded
- **THEN** `fullyParallel` SHALL be enabled
- **AND** `workers` SHALL be an explicit count in CI and derived from the machine's CPU core count locally

#### Scenario: Parallel execution safe under mock auth
- **WHEN** multiple tests execute concurrently against the same backend and test database
- **THEN** each test SHALL authenticate independently via its own `X-Test-Org-ID` header or cookie
- **AND** no test SHALL depend on a shared storage-state file or on serial execution ordering

### Requirement: Tests avoid fixed-duration sleeps

E2E tests and page objects SHALL NOT use fixed-duration sleeps (`page.waitForTimeout`) or `networkidle` waits. Synchronization SHALL use Playwright actionability auto-waiting and web-first assertions that resolve the moment a condition is met.

#### Scenario: Waiting on asynchronous UI updates
- **WHEN** a test performs an action that triggers an asynchronous UI update (e.g., search filter, create, edit, delete)
- **THEN** the test SHALL assert on the resulting UI state with a web-first assertion (e.g., `toBeVisible`, `toHaveCount`) instead of sleeping a fixed duration

#### Scenario: No fixed sleeps present in suite
- **WHEN** the suite is audited
- **THEN** no `waitForTimeout` or `networkidle` calls SHALL exist in spec files or page objects

### Requirement: Dead Playwright configuration removed

The system SHALL remove Playwright configuration that references artifacts the suite does not produce or use.

#### Scenario: No orphan storage-state config
- **WHEN** `playwright.config.ts` is loaded
- **THEN** it SHALL NOT reference a `storageState` file that the global setup never writes

#### Scenario: No unused auth page objects
- **WHEN** the E2E project is inspected
- **THEN** unused page objects (such as a mock-auth `LoginPage`) SHALL NOT remain in the project if no spec imports them

### Requirement: Mock invoicing provider server in the e2e environment

The e2e environment SHALL boot a mock invoicing provider server (Go command `cmd/mock-siigo`) that implements the Siigo adapter surface: OAuth token grant at the configured token URL, no company endpoint (404), paginated customer list, invoice creation with consecutive numbering and Idempotency-Key deduplication (same key returns the previously created invoice), and invoice status lookup. The backend SHALL be configured with `SIIGO_BASE_URL`, `SIIGO_TOKEN_URL`, and `SIIGO_WEBHOOK_SECRET` pointing at the mock; no e2e test SHALL make a network call to the real Siigo API.

#### Scenario: E2E backend talks only to the mock provider

- **WHEN** the e2e stack boots and a test drives connect, import, or invoice creation
- **THEN** all provider traffic SHALL hit the mock server
- **AND** the real Siigo API SHALL NOT be reachable from the e2e configuration

#### Scenario: Idempotency-Key honored by the mock

- **WHEN** two invoice POSTs carry the same Idempotency-Key for the same organization and deal
- **THEN** the mock SHALL return the first created invoice and SHALL NOT create a duplicate

### Requirement: Siigo test organization in the seed

The seed command SHALL create a dedicated `test-org-siigo` organization (Pro plan) with an admin account and a member account, reserved for the Siigo onboarding e2e suite, so scenario state (connection rows, imports) does not leak into the general-purpose seeded organizations.

#### Scenario: Seed provides the Siigo org and RBAC accounts

- **WHEN** `cmd/seed-e2e` runs
- **THEN** `test-org-siigo` SHALL exist with admin and member accounts
- **AND** the general-purpose seeded orgs SHALL remain available for their existing suites



### Requirement: Cross-organization data isolation is E2E-tested

The E2E tests SHALL verify that data created by one seeded organization is invisible to another seeded organization at the API level.

#### Scenario: Org A data absent from Org B list

- **WHEN** an org creates a contact under its own seeded org
- **THEN** the same contact SHALL NOT appear in another org's contacts list

### Requirement: Pagination behavior is E2E-tested

The E2E tests SHALL verify the CRM list pagination contract: default `limit` of 20, explicit `limit`/`offset` parameters, and full result retrieval beyond the default page size.

#### Scenario: Default limit returns 20 rows

- **WHEN** an org has more than 20 contacts and a list request is made without `limit`/`offset`
- **THEN** exactly 20 rows SHALL be returned

#### Scenario: Explicit limit and offset retrieve the remainder

- **WHEN** a list request specifies `limit` and `offset` beyond the first page
- **THEN** the remaining rows SHALL be returned

### Requirement: Outbound reply persistence is E2E-tested

The E2E tests SHALL verify that sending a reply via `POST /crm/conversaciones/:id/mensajes` persists an outbound message retrievable through the messages API.

#### Scenario: Reply persists as an outbound message

- **WHEN** a reply is sent to an existing conversation via the messages endpoint
- **THEN** the persisted message retrieved via `/crm/conversaciones/:id/mensajes` SHALL have `direction` equal to `outbound`

### Requirement: Mock-auth guard is E2E-tested

The E2E tests SHALL verify that, with `AUTH_MOCK_ENABLED`, a request without an `X-Test-Org-ID` header is rejected with 401.

#### Scenario: Missing mock header returns 401

- **WHEN** a request is made with `AUTH_MOCK_ENABLED=true` and no `X-Test-Org-ID` header
- **THEN** the response SHALL have status 401

### Requirement: RBAC boundary is E2E-tested

The E2E tests SHALL verify that a member without the `org:manage` permission cannot access org-management-gated endpoints, returning 403.

#### Scenario: Member access to org-manage-gated endpoint is rejected with 403

- **WHEN** a member account attempts to access an endpoint gated by the `org:manage` permission (e.g., `GET /api/v1/whatsapp/config`)
- **THEN** the response SHALL have status 403
