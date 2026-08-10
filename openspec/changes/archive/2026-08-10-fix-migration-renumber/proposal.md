## Why

`go-b2b-starter/internal/db/postgres/sqlc/migrations/` contains duplicate migration version numbers: `000002` (two independent sets), `000016` (three `.sql` files including a one-off audit script), and `000020` (two independent sets). golang-migrate rejects any source tree with duplicate versions before touching the database, so `migrate up` fails on every fresh database. This blocks `make migrateup`, the e2e job in `.github/workflows/ci.yml` (change `add-ci-pipeline`), `make test-e2e`, and the verification gates of several active OpenSpec changes.

## What Changes

- Renumber `000002_add_tenant_isolation.{up,down}.sql` → `000022_add_tenant_isolation.{up,down}.sql` (appended after `000021_create_invoices`).
- Renumber `000020_create_whatsapp_signup_flows.{up,down}.sql` → `000023_create_whatsapp_signup_flows.{up,down}.sql`.
- Relocate the operational audit/repair script `000016_pre_migration_audit.sql` out of `migrations/` into `internal/db/postgres/sqlc/audit/000016_pre_migration_audit.sql` so it is never scanned by the migration runner.
- Add a migration-consistency guard so duplicate versions cannot regress: a shell check in `scripts/` (and wired into `make migrateup`/CI) that fails if any version prefix repeats across `*.sql` files in `migrations/`.
- Verify a fresh `migrate up` on an empty `saas_db_test` now succeeds end-to-end.
- **BREAKING**: for any database already migrated to `000021` with the pre-split numbering, migration history is renumbered; those databases require `migrate force 23` (see Migration Plan in design.md). No local database has an applied `schema_migrations` history (verified: `saas_db_test`, `saas_db_e2e` both lack the table), so no local action is needed.

## Capabilities

### New Capabilities
- `migration-integrity`: unique, sequentially-ordered migration versions, migration audit scripts kept out of the scanned migration directory, and an automated guard preventing duplicate version regressions.

### Modified Capabilities
<!-- None: this does not change runtime behaviour of existing capability specs. -->

## Impact

- **Modified files** (renamed): `go-b2b-starter/internal/db/postgres/sqlc/migrations/000002_add_tenant_isolation.{up,down}.sql` → `000022_…`; `000020_create_whatsapp_signup_flows.{up,down}.sql` → `000023_…`.
- **Moved**: `000016_pre_migration_audit.sql` → `internal/db/postgres/sqlc/audit/`.
- **New**: `go-b2b-starter/scripts/check_migrations.sh` + `make check-migrations` target; CI step in `.github/workflows/ci.yml`.
- **Systems**: DB migrations (Static SSOT), local tooling, CI. No application code or API behavior changes.

## Non-Goals

- No schema changes, no new tables/columns, no SQL rewrites — pure version-number/position cleanup.
- No autopilot, no local credential storage (unchanged invariant); no change to migration content or ordering semantics beyond deduplication.
- No changes to already-applied migrations' content.

## Rollback

- **Git state**: revert the renames/moves and delete the guard script — files return to the (broken) prior state; no schema state is touched by a revert.
- **Stytch tenant policy state**: no Stytch policy changes — nothing to roll back.
- **Database state**: for a database that applied the renumbered chain, rollback = apply the matching `.down` migrations in reverse or restore from backup; for a database force-marked to `23`, `migrate force 22` restores the pre-rename history pointer.
