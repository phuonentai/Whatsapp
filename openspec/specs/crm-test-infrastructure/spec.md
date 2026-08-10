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

The system SHALL seed the test database in `global-setup.ts` with three organizations at different plan tiers and one organization for RBAC testing.

#### Scenario: Global setup seeds test orgs
- **WHEN** `global-setup.ts` runs
- **THEN** it SHALL create `test-org-free` (Free plan), `test-org-pro` (Pro plan), `test-org-enterprise` (Enterprise plan), and `test-org-rbac` (Pro plan)
- **AND** SHALL create admin accounts for each org
- **AND** SHALL create manager and member accounts for `test-org-rbac`

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
