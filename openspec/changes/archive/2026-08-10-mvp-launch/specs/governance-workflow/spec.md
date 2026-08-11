## ADDED Requirements

### Requirement: Launch change reconciles in-flight change state

A change whose purpose is launch readiness SHALL reconcile the task state and factual premises of its in-flight dependent changes against the codebase before reporting completion. Reconciliation SHALL include correcting stale migration-number claims in proposals/designs to match the on-disk migration files, and marking deferred tasks (e.g., requiring live sandbox credentials) as deferred in the owning change rather than as blockers.

#### Scenario: Migration reference corrected

- **WHEN** an in-flight change's proposal/design claims a migration number that does not match the migration file on disk
- **THEN** the launch change SHALL record a task correcting the claim to the actual on-disk number
- **AND** SHALL verify no two in-flight changes claim the same migration number that maps to different files

#### Scenario: Live-credential tasks deferred

- **WHEN** a dependent change's tasks require live provider credentials (e.g., Polar/MercadoPago sandbox) that cannot be exercised locally
- **THEN** the launch change SHALL record those tasks as deferred in the owning change
- **AND** SHALL NOT block the code-level launch gate on them

### Requirement: Launch verification gate

The launch change SHALL define and execute a single verification gate covering backend, frontend, and E2E before the change may be reported complete. The gate SHALL run in a fixed order and record results in `tasks.md`: backend (`make sqlc`, `go build ./...`, `go vet ./...`, `make test`), frontend (`pnpm lint`, `tsc --noEmit`, `pnpm build`), then E2E (`pnpm exec playwright test`).

#### Scenario: All layers pass

- **WHEN** the backend, frontend, and E2E gate commands all exit zero
- **THEN** the change SHALL be reported complete
- **AND** the results SHALL be recorded in `tasks.md`

#### Scenario: A layer fails

- **WHEN** any gate command exits non-zero
- **THEN** the change SHALL remain in-progress
- **AND** the failing command and output SHALL be recorded in `tasks.md`

### Requirement: Frontend and E2E CI before launch

The repository SHALL have frontend and E2E CI coverage before the MVP launch change is reported complete. The backend CI SHALL run with a Go image version consistent with the module's go.mod. A frontend/E2E CI job SHALL exist in addition to the existing backend job.

#### Scenario: Frontend and E2E CI jobs present

- **WHEN** the launch gate runs
- **THEN** a CI job SHALL run frontend verification (`pnpm lint`, `pnpm build`)
- **AND** a CI job SHALL run the Playwright E2E suite against test infrastructure
- **AND** the backend CI image SHALL match the go.mod Go version
