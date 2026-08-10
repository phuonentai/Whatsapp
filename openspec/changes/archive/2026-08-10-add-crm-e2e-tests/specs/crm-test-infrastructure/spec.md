## ADDED Requirements

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
