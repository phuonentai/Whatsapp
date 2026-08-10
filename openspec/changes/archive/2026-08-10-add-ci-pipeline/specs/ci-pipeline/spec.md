## ADDED Requirements

### Requirement: CI pipeline runs backend checks
The repository SHALL provide a GitHub Actions workflow (`.github/workflows/ci.yml`) that runs on pushes and pull requests and executes a `backend` job that builds the Go module and runs the unit test suite with coverage.

#### Scenario: Backend job passes on clean code
- **WHEN** a push or pull request triggers the workflow
- **THEN** the `backend` job compiles the Go module (`go build ./...`) without errors
- **AND** `make test` runs and passes with a coverage report generated

#### Scenario: Backend job fails on broken code
- **WHEN** the Go module fails to build or a unit test fails
- **THEN** the `backend` job fails
- **AND** the workflow reports failure without running dependent stages

### Requirement: CI pipeline runs frontend checks
The workflow SHALL run a `frontend` job that installs dependencies with pnpm, runs lint, and runs the production build (typecheck) of `next_b2b_starter`.

#### Scenario: Frontend job passes
- **WHEN** the `frontend` job runs
- **THEN** `pnpm lint` completes with zero errors
- **AND** `pnpm build` completes successfully

#### Scenario: Frontend job fails on lint or type error
- **WHEN** lint reports an error or the build fails to typecheck
- **THEN** the `frontend` job fails

### Requirement: CI pipeline runs offline e2e suite
The workflow SHALL run an `e2e` job that boots the full stack in a canonical offline test environment and executes the Playwright suite.

#### Scenario: E2E environment boots and tests pass
- **WHEN** the `e2e` job starts
- **THEN** PostgreSQL (pgvector) and Redis services are available
- **AND** database migrations are applied to the `saas_db_test` database
- **AND** `cmd/seed-e2e` seeds the test organizations with `AUTH_MOCK_ENABLED=true`
- **AND** the Go backend starts on port `8080` with `AUTH_MOCK_ENABLED=true`
- **AND** the Next.js frontend starts on port `3001`
- **AND** `pnpm test:e2e` runs the Playwright suite to completion

#### Scenario: E2E failure produces report artifact
- **WHEN** any Playwright test fails
- **THEN** the `e2e` job fails
- **AND** the `playwright-report/` artifact is uploaded for inspection

### Requirement: CI uses canonical e2e environment
The e2e environment SHALL use the canonical configuration: backend on port `8080`, frontend on port `3001`, and test database `saas_db_test`. Environment files and e2e helper defaults SHALL agree on these values.

#### Scenario: Port and DB configuration is consistent
- **WHEN** the workflow or `make test-e2e` runs the e2e suite
- **THEN** the backend listens on port `8080`
- **AND** the frontend is reachable at port `3001`
- **AND** the test database is named `saas_db_test`

### Requirement: CI blocks mock auth in production configuration
The workflow SHALL include a check that `AUTH_MOCK_ENABLED=true` appears only in non-production environment files.

#### Scenario: Mock auth flag found in production config
- **WHEN** the workflow checks environment files
- **THEN** `AUTH_MOCK_ENABLED=true` present in production configuration (e.g., `app.env`, production compose) fails the check
- **AND** `AUTH_MOCK_ENABLED=true` present only in `next_b2b_starter/.env.local` passes
