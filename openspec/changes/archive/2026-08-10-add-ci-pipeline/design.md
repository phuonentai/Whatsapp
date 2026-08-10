## Context

The monorepo ships two apps: `go-b2b-starter` (Go 1.25, Gin, SQLC) and `next_b2b_starter` (Next.js 16, React 19, pnpm). A Playwright suite of 61 tests across 13 specs exists but is not wired into any pipeline. The Go test runner script is stale (references a `./src` tree removed during an earlier restructure). The mock-auth e2e path (`AUTH_MOCK_ENABLED=true` + `X-Test-Org-ID` header, resolved via `stytch_org_id` = org slug) is fully offline and already exercised by `cmd/seed-e2e`, which seeds four orgs (free/pro/enterprise/rbac) with subscriptions.

Current state discovered during premise validation:
- No CI files at all (`.github/`, `.gitlab-ci.yml` absent); remote is `github.com/phuonentai/Whatsapp`.
- `run_tests_with_coverage.sh` globs `./src/**/go.mod`; the module root `go.mod` is never tested.
- `playwright.config.ts` has no `webServer`; baseURL `:3001`, CI mode toggles retries=2 + html reporter.
- Port drift: `app.env SERVER_ADDRESS=:8082`, `.env.local API_REWRITE_TARGET=:8082`, but e2e helper defaults and Next rewrites default to `:8080`; Caddy routes to `backend:8080`.
- DB-name drift: `app.env POSTGRES_DB=saas_db_e2e` vs `seed-e2e` docs `saas_db_test`.
- Mock auth gated in code at `internal/modules/auth/provider.go:77` (`EnableMockAuth` only when `AUTH_MOCK_ENABLED=true`).

## Goals / Non-Goals

**Goals:**
- Green, repeatable GitHub Actions CI with three isolated jobs.
- `make test` actually runs the Go unit suite and fails loudly on failure.
- A single canonical e2e environment: backend `:8080`, frontend `:3001`, DB `saas_db_test`.
- `make test-e2e` one-command local e2e run.
- Documented e2e setup in `DEVELOPMENT.md`.
- CI guard that mock auth can never be enabled in production config.

**Non-Goals:**
- Deploy/release pipeline, containers to registry, env promotion.
- Live-credential e2e (MP/Polar/Siigo/Meta) — deferred external in owning changes.
- CI caching optimization, parallel matrix expansion, per-spec sharding.
- Migration of dev tooling (Air, docker-compose dev) beyond the test path.

## Decisions

**D1. GitHub Actions over GitLab CI.**
Remote is GitHub; task 11.1 in `add-crm-e2e-tests` specified `.gitlab-ci.yml`, and an orphaned `go-b2b-starter/.gitlab-ci.yml` exists — but the repo lives on GitHub, so GitLab CI would never run there. GitHub-hosted runners satisfy postgres+redis+playwright needs. *Alternative rejected:* GitLab CI (no GitLab remote, no runner). The existing `.gitlab-ci.yml` is reused as a structural reference (same canonical ports `:8080`/`:3001`, `saas_db_test`, mock-auth env, playwright image). If the repo later moves, the workflow is translatable.

**D2. Three jobs, not one mega-job.**
`backend`, `frontend`, `e2e` run in parallel; `e2e` alone boots the full stack. Independent failure domains, shorter wall time. *Alternative rejected:* single job (slower, blurs failures).

**D3. `e2e` job uses GitHub Actions `services` (pgvector/redis) + direct process startup, not docker-compose.**
Services key gives healthchecks for free; backend/frontend start as background processes with `$!` pid wait/health polls — matching the documented local flow (backend `:8080`, Next `:3001`). Compose-in-CI adds a layer that obscures port/env and complicates artifacts. *Alternative considered:* `docker compose -f docker-compose.yml up` — rejected: compose hardcodes `:3000`/`app.env` volumes and Stytch secrets; e2e needs mock-auth env and distinct ports.

**D4. Canonical e2e env = backend `:8080`, frontend `:3001`, DB `saas_db_test`.**
Matches `.gitlab-ci.yml`, Next rewrites, Caddy target, and seed-e2e docs. Fix the drift: `app.env` POSTGRES_DB, `.env.local` targets, and the `e2e/helpers/api.ts` default (`:8082` → `:8080`). `SERVER_ADDRESS` in `app.env` stays `:8082` as the local-dev default; CI and `make test-e2e` override via `SERVER_ADDRESS=:8080`. *Alternative rejected:* canonicalize to `:8082` (would require editing rewrites and Caddy — larger blast radius).

**D4a. Browser `/api/auth/*` proxy gap is a test-blocking bug, fixed with a rewrite.**
`next.config.ts` rewrites only `/api/crm` and `/api/v1`; the frontend owns only `app/api/auth/session/refresh`. Browser calls to `/api/auth/profile/me` and `/api/auth/members` (driven by `useProfileQuery`/`useMembersQuery` on the settings page) hit a Next.js 404, so every settings/admin view renders the profile-error block and the admin-panel e2e specs time out. Fix: add `{ source: "/api/auth/:path*", destination: "${API_REWRITE_TARGET || http://localhost:8080}/api/auth/:path*" }` using the default `afterFiles` precedence, which keeps the local `app/api/auth/session/refresh` route handler winning over the rewrite. Backend mock auth already accepts the `X-Test-Org-ID` cookie (`middleware.go:154-159`), so the browser cookie flow works unchanged.

**D5. Rewrite `run_tests_with_coverage.sh` in place.**
Keep coverage aggregation but target the module root: `go test -coverprofile ./...` run at `go-b2b-starter/`. Minimal diff, keeps existing coverage file layout so `coverage/coverage.html` consumers don't break. *Alternative rejected:* replace Makefile `test` with plain `go test ./...` (loses the coverage artifact convention).

**D6. `make test-e2e` = migrate → seed → start servers → run, with trap cleanup.**
One target: `migrateup` against `saas_db_test` env vars, `go run ./cmd/seed-e2e` with `AUTH_MOCK_ENABLED=true`, start backend (background) + `pnpm --dir ../next_b2b_starter dev -p 3001` (background), poll health, `pnpm test:e2e`, kill pids in `trap EXIT`. Local parity with the CI `e2e` job so `make test-e2e` and CI can't drift.

**D7. Mock-auth prod guard as a CI job step.**
Add a step (backend or a tiny dedicated job) that asserts `AUTH_MOCK_ENABLED=true` only appears in `next_b2b_starter/.env.local` (and not `app.env`/prod compose). Simple grep-based check; no secret handling. This closes task 15.2 (`add-crm-e2e-tests`) without runtime changes.

**D8. Playwright run uses existing config; CI only sets env vars.**
`playwright.config.ts` already switches on `process.env.CI` (retries 2, html reporter). Workflow sets `BASE_URL`/`NEXT_PUBLIC_API_URL` defaults via env and uploads `playwright-report/`. No config file change needed.

## Risks / Trade-offs

- **Port override drift (`app.env :8082` vs CI `:8080`)** → CI pins `SERVER_ADDRESS=:8080` explicitly; `make test-e2e` passes the same override; documented in `DEVELOPMENT.md` table.
- **Flaky e2e (61 serial tests)** → CI retries=2 via existing config; job is serial by design (`workers: 1`); report artifact preserves failure evidence.
- **Playwright browser install in CI** → use `pnpm exec playwright install --with-deps chromium` (needs apt deps on ubuntu runner); pinned via existing `@playwright/test` devDependency.
- **Migrate/seed not idempotent on rerun** → CI creates a fresh `saas_db_test` per run (drop/recreate or unique service DB); `make test-e2e` docs note local DB reuse caveat.
- **New workflow blockers other changes** → workflow is the verification target for tasks in `add-crm-e2e-tests` (11.1, 15.x); keep e2e job green or those changes stay in-progress.

## Migration Plan

1. Land tooling fixes (script, env canonicalization, Makefile) first — verified by existing local commands.
2. Land `.github/workflows/ci.yml` — verified by pushing to GitHub and observing the three jobs.
3. Add `make test-e2e` + `DEVELOPMENT.md` docs, verifying locally against a local postgres/redis.
4. On failure of any job, fix within this change; do not disable the workflow.

Rollback: revert commits (all additive/config-only), or delete `.github/workflows/ci.yml` if the pipeline blocks other work.

## Open Questions

- **Pre-existing migration blocker**: `migrations/` contains duplicate version numbers (`000002`, `000016`, `000020`), so golang-migrate rejects the source and fresh `migrate up` fails everywhere (local, `make migrateup`, and the CI e2e job). Fix is tracked as a separate OpenSpec change (renumber + relocate `000016_pre_migration_audit.sql` out of `migrations/`); the CI e2e job is expected to go green once it lands.
- Candidate follow-up: whether to add nightly full-suite run to keep e2e from bit-rotting between PRs (out of scope here).
