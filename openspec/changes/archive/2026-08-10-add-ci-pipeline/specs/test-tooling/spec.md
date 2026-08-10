## ADDED Requirements

### Requirement: make test runs the Go test suite
The `make test` target SHALL execute the root Go module's unit tests with coverage, using the existing coverage aggregation and report generation in `scripts/run_tests_with_coverage.sh`, and SHALL exit non-zero when any test fails.

#### Scenario: make test on clean module
- **WHEN** a developer runs `make test` in `go-b2b-starter/`
- **THEN** all root module unit tests execute
- **AND** a coverage report is generated at `coverage/coverage.html`
- **AND** the command exits with status 0

#### Scenario: make test with failing test
- **WHEN** a unit test fails
- **THEN** `make test` exits non-zero
- **AND** the failure is reported rather than silently skipped

### Requirement: make test-e2e runs the offline e2e suite locally
The `make test-e2e` target SHALL provide a one-command local run of the Playwright suite against the canonical offline environment (backend `:8080`, frontend `:3001`, DB `saas_db_test`), cleaning up spawned server processes on exit.

#### Scenario: Successful local e2e run
- **WHEN** a developer runs `make test-e2e`
- **THEN** migrations are applied to `saas_db_test`
- **AND** `cmd/seed-e2e` seeds the test organizations
- **AND** the backend starts on `:8080` and the frontend starts on `:3001`
- **AND** the Playwright suite runs to completion
- **AND** all spawned server processes are terminated on exit

#### Scenario: E2E run against unavailable database
- **WHEN** PostgreSQL or Redis is not reachable
- **THEN** `make test-e2e` fails with a clear error
- **AND** no server processes are left running

### Requirement: e2e setup is documented
The `DEVELOPMENT.md` SHALL document how to run the e2e suite, including the canonical ports, test database name, mock-auth usage, and the `make test-e2e` and CI commands.

#### Scenario: Developer reads e2e documentation
- **WHEN** a developer follows `DEVELOPMENT.md` e2e section
- **THEN** they can start the test environment and run `make test-e2e` or the Playwright suite against CI-equivalent configuration

### Requirement: browser API calls reach the backend for e2e views
The frontend SHALL proxy browser API calls to the Go backend for every path the settings/admin views depend on — including `/api/auth/:path*` (profile, members, invite, member role) — so pages render in the browser under mock auth instead of falling through to a Next.js 404. The proxy SHALL preserve the frontend's own `/api/auth/session/refresh` route handler.

#### Scenario: Profile fetch succeeds through the frontend proxy
- **WHEN** a browser session under mock auth requests `GET /api/auth/profile/me` through the frontend
- **THEN** the response status is 200 (or a backend-originated error)
- **AND** the settings page renders a view instead of the profile-error block

#### Scenario: Frontend session refresh route is preserved
- **WHEN** a browser session requests `POST /api/auth/session/refresh`
- **THEN** the request is handled by the frontend's own route handler, not the backend proxy
