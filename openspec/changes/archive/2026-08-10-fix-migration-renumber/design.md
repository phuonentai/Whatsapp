## Context

`go-b2b-starter/internal/db/postgres/sqlc/migrations/` currently has 21 logical versions but 24 `.sql` files because three versions are duplicated:

| Version | Files | Problem |
|---|---|---|
| `000002` | `create_organizations_schema.{up,down}`, `add_tenant_isolation.{up,down}` | two independent migrations share one version |
| `000016` | `create_crm_integrity_constraints.{up,down}`, `pre_migration_audit.sql` | one-off audit/repair script sits in the scanned dir |
| `000020` | `create_playbooks.{up,down}`, `create_whatsapp_signup_flows.{up,down}` | two independent migrations share one version |

golang-migrate (v4.18.2) validates the source before connecting and refuses duplicate versions with `duplicate migration file`, so fresh `migrate up` fails deterministically. This is the blocker surfaced by change `add-ci-pipeline` (verification gate 5.4). Verified premises:

- `000002_add_tenant_isolation` alters `file_manager.file_assets` and references `public.organizations(id)` — both exist by `000002_create_organizations_schema` and `000001_create_file_manager_schema`. Later migrations (`000006`, `000008`, `000021`) reference only `file_assets(id)`, never `organization_id` — so appending it at `000022` is safe.
- `000020_create_whatsapp_signup_flows` creates `whatsapp.signup_flows` referencing `organizations.organizations(id)`; nothing later references it — appending at `000023` is safe.
- No local database (`saas_db_test`, `saas_db_e2e`) has a `schema_migrations` table — nothing migrated via golang-migrate locally; fresh everywhere.

## Goals / Non-Goals

**Goals:**
- Fresh `migrate up` succeeds on an empty database (`saas_db_test` canonical).
- Uniqueness + monotonic ordering of migration versions is guaranteed and guarded against regression in CI.
- Audit/repair scripts are impossible for the migration runner to scan.
- Unblock e2e verification in `add-ci-pipeline` and `make test-e2e`.

**Non-Goals:**
- Alter migration SQL content, add/remove tables/columns, or change the order in which tables are created relative to one another.
- Fix `000002` vs `000002` "which came first" semantics beyond appending — both files keep their exact contents; only version numbers change.

## Decisions

**D1. Append the orphans, don't renumber the spine.**
Keep `000002_create_organizations_schema`, `000016_create_crm_integrity_constraints`, `000020_create_playbooks` at their current numbers (they form the logical spine other migrations assume) and renumber the two extras to `000022`/`000023`. *Alternative rejected:* full cascade renumber (rewrite every version, huge diff, high risk, no benefit since migrate only needs uniqueness + order).

**D2. Relocate `000016_pre_migration_audit.sql`, don't rename-in-place.**
Move to `internal/db/postgres/sqlc/audit/`. It is an operational repair runbook, not a migration. Keeping it in `migrations/` under any version number risks future confusion. *Alternative rejected:* delete (loses historical context).

**D3. Guard duplicate versions via a shell script wired into `make migrateup` + CI.**
`scripts/check_migrations.sh` fails if `ls migrations/ | sed 's/_.*//' | sort | uniq -d` is non-empty (ignoring non-`.sql` files). Added to the `migrateup`/`migratedown` prerequisites and as a step in `.github/workflows/ci.yml`. *Alternative rejected:* rely on golang-migrate's own error (fails too late, only when someone runs a migrate).

**D4. Migration plan for already-migrated databases.**
Any database that applied the old numbering (single `000002`, single `000020`) will see `000022`/`000023` as new and fail (duplicate column/table) when re-running `migrate up`. These DBs run `migrate force 23` (table/column already present from the original split). Local DBs need nothing (verified absent). No production DB is known to exist; if one appears, force + manual verification applies.

## Risks / Trade-offs

- **Already-migrated DB drift** → documented `migrate force 23` procedure; none known locally; CI e2e always uses a fresh DB so it is unaffected.
- **Ordering surprise**: `add_tenant_isolation` now applies after `000021` instead of at `000002` → verified no later migration references the column; for fresh DBs the outcome is identical.
- **Latent content bug (found during verification)**: `000002_add_tenant_isolation.up.sql` referenced `REFERENCES public.organizations(id)` but the table is `organizations.organizations`; it could never have applied. Corrected to the schema-qualified reference. Deviation from the "no SQL rewrites" non-goal — a one-line correction required for the migration to apply.
- **Guard false positives** → script ignores non-`.sql` files (e.g., `000016_audit_report.md`) and only checks the numeric prefix.

## Migration Plan

1. Apply file renames/moves (git mv) + guard script + Makefile + CI step.
2. Verify locally on a fresh DB: create `saas_db_test`, run `/tmp/opencode/migrate … up`, confirm all 23 migrations apply and `schema_migrations` lists 000001..000023.
3. For any pre-existing migrated DB: `migrate force 23` (content already present) — none known locally.
4. Run the e2e flow from `add-ci-pipeline` (`make test-e2e`) to confirm the blocker is cleared.

## Open Questions

- Does a production/staging database exist with applied migration history? Not observable from this environment; if yes it must run `migrate force 23` before the next deploy.
