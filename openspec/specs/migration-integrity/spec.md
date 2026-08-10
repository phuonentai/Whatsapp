# migration-integrity Specification

## Purpose
Guarantees SQL migration integrity: unique numeric version prefixes, a duplicate-version guard enforced in CI, and a renumbered tree that applies cleanly to a fresh database.
## Requirements
### Requirement: Migration versions are unique
The migrations directory `go-b2b-starter/internal/db/postgres/sqlc/migrations/` SHALL contain at most one migration set (one `.up.sql` + one `.down.sql`) per numeric version prefix. No two `.sql` files in the directory SHALL share the same leading version number.

#### Scenario: Fresh database applies the full chain
- **WHEN** `migrate -path internal/db/postgres/sqlc/migrations -database <fresh-db> up` runs
- **THEN** all migrations apply in ascending version order without a duplicate-version error
- **AND** `schema_migrations` records versions `000001` through `000023`

#### Scenario: Duplicate version present
- **WHEN** a `.sql` migration file shares its numeric version prefix with another `.sql` file in `migrations/`
- **THEN** the guard check (`scripts/check_migrations.sh` / `make check-migrations`) fails

### Requirement: Migration audit scripts are kept out of the scanned directory
Operational one-off scripts that are not migrations SHALL NOT reside in `migrations/`. The `000016_pre_migration_audit.sql` repair runbook SHALL live under `internal/db/postgres/sqlc/audit/`.

#### Scenario: Migration runner ignores audit scripts
- **WHEN** `migrate -path internal/db/postgres/sqlc/migrations up` scans the directory
- **THEN** no audit/repair scripts are treated as migrations
- **AND** the audit script remains available for reference under `internal/db/postgres/sqlc/audit/`

### Requirement: Migration guard runs in local tooling and CI
The duplicate-version guard SHALL run as a prerequisite of the migrate targets in `go-b2b-starter/Makefile` and as a step in `.github/workflows/ci.yml`.

#### Scenario: Guard runs on migrate
- **WHEN** a developer runs `make migrateup` or `make migratedown`
- **THEN** `scripts/check_migrations.sh` executes first and fails the command on duplicate versions

#### Scenario: Guard runs in CI
- **WHEN** the CI workflow runs
- **THEN** a step executes `scripts/check_migrations.sh`
- **AND** the workflow fails if the migrations directory contains duplicate versions

