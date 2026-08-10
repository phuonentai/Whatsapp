## ADDED Requirements

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
