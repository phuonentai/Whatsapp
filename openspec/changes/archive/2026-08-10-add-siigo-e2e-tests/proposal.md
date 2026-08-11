## Why

The Siigo onboarding build (changes `add-siigo-org-onboarding`, `add-siigo-onboarding-data`, `add-siigo-onboarding-wizard` — all complete, all with archive deferred pending sandbox verification) shipped a full connect→numeraciön→import→sandbox→activate state machine, gating, import, and wizard UX with zero end-to-end coverage. The e2e runner (`scripts/run_e2e.sh`) boots the backend with no `SIIGO_*` environment at all, so the wizard would call the real Siigo API with placeholder credentials and fail. Backend test coverage also has holes (test-invoice service, nightly delta job, admin connection list, provider resolver all untested), and no Playwright spec exercises the onboarding flow. Enterprise-grade delivery of this build requires: a mock Siigo provider server (offline, deterministic), backend unit + DB-backed integration tests for the gaps, and Playwright e2e specs covering the wizard, kill-switch, assisted setup, admin view, deal-stage gating, and cross-org isolation.

## What Changes

- **Mock Siigo provider server**: new Go command `go-b2b-starter/cmd/mock-siigo` implementing the adapter surface (OAuth token grant, `GET /v1/company` → 404, paginated `GET /v1/customers`, `POST /v1/invoices` with Idempotency-Key dedupe + consecutive numbering, `GET /v1/invoices/{id}`, optional signed webhook delivery). Booted by `scripts/run_e2e.sh` with `SIIGO_BASE_URL`/`SIIGO_TOKEN_URL`/`SIIGO_WEBHOOK_SECRET` pointing at it.
- **e2e wiring**: `run_e2e.sh` starts mock-siigo on `:8090` (healthy before backend boot), exports `SIIGO_*` envs for the backend; `seed-e2e` gains a dedicated `test-org-siigo` org (Pro plan) plus a second RBAC member account so member-vs-admin permission scenarios are testable.
- **Connection-state isolation**: `seed-e2e` may optionally reset/delete invoicing connection rows per run (DB is dropped per run already — the mock customers table also resets); no production reset endpoint is added.
- **Backend test gaps closed**: unit tests for `TestInvoiceService`, `runDeltaSyncOnce` (nightly job), `AdminListConnections` handler, `ConnectionProviderResolver`, and adapter NIT-mismatch-with-provider-data paths; DB-backed integration tests (existing `internal/db/postgres/sqlc/integration` harness) for connection repository CRUD + state guard and import confirm idempotency against the real schema.
- **Playwright e2e specs**: new `e2e/specs/siigo-onboarding.spec.ts` (+ page-object additions to `settings.page.ts` and `admin-panel.page.ts`) covering: wizard happy path (connect→numeración→import preview/confirm→test invoice→activate), kill-switch pause/resume, assisted setup (request → admin provisions → org reaches connected), admin onboarding table + provision form, deal→facturado gating (no invoice without `live`; invoice + notification with `live`), and org isolation (a `live` org does not affect a fresh org).

## Capabilities

### New Capabilities
- `siigo-onboarding-e2e`: Playwright scenarios for the Siigo onboarding wizard, kill-switch, assisted setup, admin view, deal-stage gating, and org isolation.

### Modified Capabilities
- `crm-test-infrastructure`: the canonical e2e environment SHALL boot a mock invoicing provider server (no real Siigo network calls) and SHALL include a Siigo test organization + RBAC accounts in the seed.
- `test-tooling`: the `make test-e2e` bootstrap SHALL start/stop the mock Siigo server and export its `SIIGO_*` configuration to the backend.

## Impact

- **Go backend**: new `cmd/mock-siigo` (test-only binary, never wired into production routes); `cmd/seed-e2e` adds `test-org-siigo` org + accounts; no production code changes required unless a test surfaces a defect (then the fix lands in the owning change's scope with a note).
- **e2e runner**: `scripts/run_e2e.sh` (boot/teardown mock + env export); `DEVELOPMENT.md` e2e section gains the mock provider note (via `test-tooling` requirements).
- **Frontend**: `e2e/page-objects/settings.page.ts` + `admin-panel.page.ts` extensions and `e2e/specs/siigo-onboarding.spec.ts`; `e2e/helpers/api.ts` reuse; no production FE changes.
- **Tests**: new Go unit tests in `internal/modules/invoicing/...`; new integration tests under `internal/db/postgres/sqlc/integration/` (DB harness pattern).
- **Auth boundary / Stytch**: untouched — mock auth (`AUTH_MOCK_ENABLED` + `X-Test-Org-ID`) is already the e2e contract; no real credentials anywhere.
- **Dependencies**: none new (mock server uses stdlib net/http only).
- **Rollback strategy**: Git — revert the change commit(s); DB — no migration (seed-only changes); Stytch tenant policy — unaffected. Mock server and seed additions are test-only artifacts.

## Assumptions

- The Siigo adapter contract implemented by `cmd/mock-siigo` (token grant at `SIIGO_TOKEN_URL`, `/v1/company` absent → 404, `/v1/customers?page=&page_size=100`, `/v1/invoices` POST/GET, `Idempotency-Key` echo) matches the spike-verified contract recorded in `add-siigo-onboarding-data` task 1.1; the mock is authoritative for offline tests and can be re-aligned at deployment-time sandbox verification without touching the specs.
- The integration-test DB harness (`internal/db/postgres/sqlc/integration/harness_test.go`) is the intended vehicle for DB-backed tests; it currently contains no invoicing references (verified) and is extended rather than replaced.
- A real WhatsApp outbound send in e2e (invoice notification) is already exercised by the existing whatsapp e2e pattern (signed webhook ingress + config health assertions); the Siigo e2e asserts the notification path at the webhook/invoice-status level, not against a live Meta number.

## Non-Goals

- No production reset/delete endpoint for connections (state isolation is solved at seed level).
- No load/soak testing of the nightly delta sync (covered by unit test; a soak harness is out of scope).
- No mocking at the Playwright route level (backend must call the real mock server over HTTP so the full stack is exercised).
- No changes to the Siigo adapter, state machine, or wizard production code unless a test uncovers a defect.
- No new vendor test frameworks (Go stdlib + Playwright + vitest as configured).
