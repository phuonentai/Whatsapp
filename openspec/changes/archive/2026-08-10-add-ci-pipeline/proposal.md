## Why

ROADMAP.md declares "E2E suite + CI matrix" as shipped, but the repo has zero CI: no `.github/workflows/`, no `.gitlab-ci.yml`. The Go test target is silently broken (`run_tests_with_coverage.sh` globs a `./src/` tree that no longer exists, so `make test` exits 0 having run nothing), the Playwright suite (61 tests across 13 specs) has no `webServer` orchestration or `make test-e2e` entry point, and e2e port/DB configuration has drifted (`.env.local` comments say backend `:8080` but set `:8082`; `app.env` uses `saas_db_e2e` while `cmd/seed-e2e` documents `saas_db_test`). Without CI and a working local e2e path, regressions land silently and the verification gates of several active OpenSpec changes cannot be executed.

## What Changes

- Add `.github/workflows/ci.yml` with three jobs: `backend` (build + unit tests + coverage), `frontend` (typecheck + lint), `e2e` (spin pgvector+redis → migrate → seed-e2e → start backend on `:8080` → start Next.js on `:3001` → run Playwright → upload report artifact).
- Fix `go-b2b-starter/scripts/run_tests_with_coverage.sh` so `make test` runs the root Go module instead of the nonexistent `./src` tree.
- Canonicalize the e2e environment: backend `:8080`, frontend `:3001`, test database `saas_db_test`. Fix `next_b2b_starter/.env.local` port values, `go-b2b-starter/app.env` DB name, and e2e helper defaults (`e2e/helpers/api.ts`, `e2e/helpers/whatsapp.ts`).
- Close the browser API proxy gap: `next_b2b_starter/next.config.ts` rewrites only `/api/crm` and `/api/v1`, so browser calls to `/api/auth/*` (profile, members, invite, member role) fall through to a Next.js 404 — the frontend owns only `app/api/auth/session/refresh`. Add an `/api/auth/:path*` rewrite (default `afterFiles`, so the local session-refresh route handler keeps priority) so `/api/auth/profile/me` and `/api/auth/members` reach the Go backend and the settings/admin views render in the browser during e2e.
- Add `make test-e2e` target in `go-b2b-starter/Makefile`: migrate → seed → start both servers → run Playwright.
- Add e2e test setup documentation to `DEVELOPMENT.md`.
- Add a CI check that `AUTH_MOCK_ENABLED=true` appears only in non-production env files, closing the mock-auth production reachability gap.
- These changes affect only tooling and environment config. No application code, API behavior, or data persistence changes.

## Capabilities

### New Capabilities
- `ci-pipeline`: GitHub Actions continuous integration — backend/frontend checks and the offline e2e suite (canonical ports `:8080`/`:3001`, DB `saas_db_test`, mock-auth only, report artifact upload).
- `test-tooling`: working `make test` (root module coverage), `make test-e2e` entry point, and e2e setup documentation in `DEVELOPMENT.md`.

### Modified Capabilities
<!-- None: no requirement changes to existing capability specs (governance-workflow, whatsapp-*, etc.). -->

## Impact

- **New files**: `.github/workflows/ci.yml`.
- **Modified files**: `go-b2b-starter/Makefile`, `go-b2b-starter/scripts/run_tests_with_coverage.sh`, `go-b2b-starter/app.env` (POSTGRES_DB), `next_b2b_starter/.env.local`, `next_b2b_starter/e2e/helpers/api.ts`, `next_b2b_starter/e2e/helpers/whatsapp.ts`, `DEVELOPMENT.md`.
- **Systems**: CI/CD (GitHub Actions), local dev/test tooling. No API surface, DB schema, or runtime auth behavior changes.
- **Dependencies**: `pgvector/pgvector:pg16`, `redis:alpine`, `migrate/migrate:v4.18.2` (already used by `docker-compose.yml`), `@playwright/test` (already a devDependency), Go 1.25, pnpm.

## Non-Goals

- No deploy/release pipeline, artifact registry, or environment promotion.
- No live-credential e2e: MercadoPago/Polar/Siigo sandbox tests and Meta WhatsApp embedded-signup smoke remain deferred (external credentials) in their owning changes.
- No CI caching/performance optimization beyond what is trivial.
- No local credential storage: this change adds no secret persistence; all secrets pass via GitHub Actions secrets / repo env files already in place.

## Rollback

- **Git state**: delete `.github/workflows/ci.yml`; revert Makefile, script, env files, and DEVELOPMENT.md edits. All changes are additive or config-only, so rollback is a clean revert.
- **Stytch tenant policy state**: no Stytch policy changes exist for this change — nothing to roll back.
