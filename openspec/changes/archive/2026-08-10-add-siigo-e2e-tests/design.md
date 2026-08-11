## Context

The three Siigo onboarding changes are complete (state machine + connect/NIT + import + numeration + test-invoice + gating + wizard + admin view). Test surface today: Go unit tests cover secrets, adapter, router, connection service, invoicing service + listener, numeration, import, and handlers; FE vitest covers the section (9) and admin view (3). Gaps: no Playwright e2e, no mock provider, no DB-backed integration tests, and untested `TestInvoiceService`, nightly `runDeltaSyncOnce`, `AdminListConnections`, `ConnectionProviderResolver`.

Verified repo facts: `scripts/run_e2e.sh` boots backend `:8080` + frontend `:3001` with `AUTH_MOCK_ENABLED=true`, drops/creates `saas_db_test`, applies migrations, runs `cmd/seed-e2e` (orgs free/pro/enterprise/rbac), no `SIIGO_*` envs today. Mock auth = `X-Test-Org-ID: <slug>:<email>`. Playwright suite lives in `next_b2b_starter/e2e/` (page-objects, helpers/api.ts with `apiRequest`, specs). DB harness at `internal/db/postgres/sqlc/integration/harness_test.go` (no invoicing coverage). The adapter's spike-verified contract (auto numbering, Idempotency-Key, no `/v1/company`, `GET /v1/customers?page=&page_size=100`) is the mock's spec.

## Goals / Non-Goals

**Goals:**
- Offline, deterministic e2e for the full Siigo onboarding surface (no real Siigo network calls)
- Close the four backend test gaps + DB-backed integration coverage
- Playwright specs: wizard happy path, kill-switch, assisted setup, admin view, gating, isolation
- Canonical runner + seed extended so the suite is one command (`make test-e2e`)
- Spec deltas capturing the mock-provider and seed contracts

**Non-Goals:**
- No production reset endpoint; no load/soak tests; no Playwright route-level mocking; no production code changes unless a defect surfaces; no new frameworks

## Decisions

### 1. Mock provider as a Go command, not a test-only backend flag

**Chosen:** `go-b2b-starter/cmd/mock-siigo` — a standalone Go binary (stdlib `net/http`) run by `run_e2e.sh` on `:8090`. Backend points at it purely via env (`SIIGO_BASE_URL`, `SIIGO_TOKEN_URL`); the production adapter is untouched and the e2e stack exercises the real HTTP path.

**Alternatives considered:**
- *Backend `SIIGO_MOCK=true` flag with in-process fake* — rejected: bypasses HTTP, risks diverging from the real adapter surface, and pollutes production config surface.
- *Playwright `page.route` interception* — rejected: only covers the browser→backend leg; backend→provider calls would still escape to the network.
- *Node/Express mock* — rejected: adds a second language to the toolchain; Go stdlib mock shares types with the adapter contract.

### 2. Mock behavior mirrors the spike-verified adapter contract

**Chosen:** token grant at `SIIGO_TOKEN_URL` (returns `access_token`); `GET /v1/company` → 404 (spike: endpoint does not exist — also keeps the connect NIT tolerance path exercised); `GET /v1/customers?page={n}&page_size=100` (fixed in-memory roster, page 0-based, short page terminates pagination); `POST /v1/invoices` assigns consecutive numbers (`FAC1-00001…`) and dedupes by `Idempotency-Key` (per org+deal map → return first invoice); `GET /v1/invoices/{id}` returns status (`valid` for the seeded fixture invoice after a short delay or immediately — deterministic flag). In-memory state resets on process restart (DB is dropped per run anyway).

**Alternatives considered:**
- *Configurable fixtures per scenario* — rejected for MVP: a single deterministic roster suffices; scenario variety is driven by the CRM/connection state, not provider data.
- *Real sandbox in CI* — rejected: external dependency + credentials in CI violates the offline contract.

### 3. Dedicated `test-org-siigo` + member account in the seed

**Chosen:** `cmd/seed-e2e` adds `test-org-siigo` (Pro plan) with `admin-siigo@test.com` and `member-siigo@test.com`. The wizard/gating e2e scenarios run against this org so connection/import state never pollutes free/pro/enterprise/rbac suites (which run earlier in the same DB).

**Alternatives considered:**
- *Reuse test-org-pro* — rejected: its suite asserts plan-gated UI; leftover connection rows would leak into unrelated assertions.
- *Reset endpoint* — rejected (Non-Goal): seed-level isolation is sufficient and production-safe.

### 4. Backend gap tests: unit first, integration second

**Chosen:** Unit tests for `TestInvoiceService` (mock provider + mock connection service: pending stays pending, valid advances to sandbox_ok, failed status check is non-fatal), `runDeltaSyncOnce` (mock import service + repo returning live orgs; failures logged not fatal), `AdminListConnections` (stub services/repos asserting aggregation shape), `ConnectionProviderResolver` (live→siigo, other→none, missing→none), and adapter `ValidateCredentials` company-present NIT paths. Integration tests (harness) cover: connection repo CRUD + state-guard (DB CHECK), and import confirm idempotency (second confirm creates no duplicates) against the real schema.

**Alternatives considered:**
- *Integration-only* — rejected: the harness needs a live DB (not always available in this environment); unit tests keep the gate runnable offline.

### 5. Playwright: one spec file + page-object extensions

**Chosen:** `e2e/specs/siigo-onboarding.spec.ts` (describe blocks per scenario group) with `settings.page.ts` gaining `openSiigoSection()`, `connectSiigo(creds)`, `confirmNumeration()`, `previewImport()/confirmImport()`, `testInvoice()`, `activateInvoicing()`, `toggleKillSwitch()`; `admin-panel.page.ts` gaining the onboarding table + provision helpers; `apiRequest` reused for deal/webhook fixtures. All assertions web-first (no `waitForTimeout` — `crm-test-infrastructure` rule).

**Alternatives considered:**
- *Separate spec per scenario* — rejected: shared onboarding state (happy path drives later gating asserts in one describe with `test.describe.serial`) keeps runtime sane; the isolation scenario uses a second org.

### 6. e2e runner wiring

**Chosen:** `run_e2e.sh`: build+start mock-siigo (`go run ./cmd/mock-siigo -addr :8090`), wait for `/healthz`, export `SIIGO_BASE_URL=http://localhost:8090`, `SIIGO_TOKEN_URL=http://localhost:8090/token`, `SIIGO_WEBHOOK_SECRET=test_webhook_secret_for_e2e` before backend boot; kill in `cleanup()` with the other PIDs. `DEVELOPMENT.md` gains a short note.

**Alternatives considered:**
- *Compose service* — rejected: the runner already manages processes directly; a compose service adds Docker coupling for a test-only binary.

## Risks / Trade-offs

- [Mock diverges from real Siigo contract] → Mitigation: mock mirrors the spike-verified contract; a single `contract.md`-style comment header in the mock documents each endpoint against its spike finding; re-align at deployment sandbox verification.
- [Serial e2e state coupling (happy path before gating/live scenarios)] → Mitigation: `test.describe.serial` within the spec; each scenario cleans up only what it creates (deals), connection state intentionally persists within the org.
- [Harness integration tests need live DB] → Mitigation: skipped cleanly when `TEST_DATABASE_URL` unset (harness precedent), unit tests keep offline coverage.
- [seed-e2e change affects existing suites] → Mitigation: purely additive org/accounts; existing orgs untouched; verify full Playwright suite still passes in the gate.
- [run_e2e.sh growth (third process)] → Mitigation: mirrors the existing backend/frontend start+cleanup pattern exactly; the mock has no external deps.

## Migration Plan

1. `cmd/mock-siigo` (server + fixtures + health endpoint) with Go unit tests for its own behavior (token grant, pagination, numbering, Idempotency-Key dedupe).
2. `scripts/run_e2e.sh` wiring + `cmd/seed-e2e` org additions + `DEVELOPMENT.md` note.
3. Backend unit gap tests + integration tests (harness).
4. FE page-object extensions + `siigo-onboarding.spec.ts`.
5. Gate: `go test ./...` (unit), `go build ./...`, `npx tsc --noEmit`, `pnpm build`, `pnpm lint`, `make test-e2e` (full Playwright suite incl. new spec).
6. Rollback: revert commit(s); no migration; no Stytch state.

## Open Questions

- Should the mock's invoice status transition to `valid` immediately or after a small deterministic delay (exercises the polling fallback in e2e)? (default: immediate for wizard speed; a `-delay-ms` flag covers polling if needed)
- Should the e2e gating scenario assert the WhatsApp notification path via the existing webhook-based inbox pattern, or only the invoice status? (default: invoice status only — live Meta send is a deployment-step deferral)
