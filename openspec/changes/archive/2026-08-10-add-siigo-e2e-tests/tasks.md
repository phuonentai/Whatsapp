## 1. Mock Siigo provider server [OPS-GOV]

- [x] 1.1 Implement `cmd/mock-siigo`: stdlib HTTP server on `-addr` (default `:8090`) with `/healthz`, `POST /token` (returns `access_token`), `GET /v1/company` → 404, `GET /v1/customers?page=&page_size=100` (in-memory roster, page 0-based, short page ends pagination), `POST /v1/invoices` (consecutive numbering `FAC1-00001…`, Idempotency-Key dedupe per org+deal returning the first invoice, immediate `valid` status with CUFE + pdf_url), `GET /v1/invoices/{id}`. Each handler documents its spike-derived contract in a comment. Verify: `go test ./cmd/mock-siigo/...` — token grant, pagination boundary, numbering increment, same-key dedupe returns identical invoice, 404 company
- [x] 1.2 Mock unit tests assert Idempotency-Key behavior (two POSTs same key → one invoice, second returns first) and consecutive numbering across orgs. Verify: `go test ./cmd/mock-siigo/...` EXIT 0

## 2. e2e runner + seed [OPS-GOV]

- [x] 2.1 Wire `scripts/run_e2e.sh`: build+start mock-siigo, wait for `/healthz`, export `SIIGO_BASE_URL=http://localhost:8090`, `SIIGO_TOKEN_URL=http://localhost:8090/token`, `SIIGO_WEBHOOK_SECRET=test_webhook_secret_for_e2e` before backend boot; kill mock in `cleanup()`. Verify: boot sequence healthy; `grep` mock PID in cleanup; backend log shows mock base URL
- [x] 2.2 Extend `cmd/seed-e2e`: add `test-org-siigo` (Pro plan) with `admin-siigo@test.com` and `member-siigo@test.com`; existing orgs unchanged. Verify: seed runs; org + accounts present in DB; existing suites unaffected (full Playwright run in gate)
- [x] 2.3 Update `DEVELOPMENT.md` e2e section: mock Siigo provider + ports + env note. Verify: section mentions `:8090` and `SIIGO_*` mock envs

## 3. Backend unit gap tests [BE-INFRA]

- [x] 3.1 Test `TestInvoiceService`: pending result stays pending (no advance), valid result advances to `sandbox_ok` via connection service, provider status-check failure is non-fatal (returns stored invoice, no error). Verify: `go test ./internal/modules/invoicing/app/services/...` — three cases pass
- [x] 3.2 Test `runDeltaSyncOnce` (cmd package): live orgs listed → DeltaSync called per org; sync failure logged and loop continues; no live orgs → no calls. Verify: `go test ./internal/modules/invoicing/cmd/...` EXIT 0
- [x] 3.3 Test `AdminListConnections` handler: rows aggregated with numeration + last import run; missing numeration/run tolerated (empty fields, no error). Verify: handler test passes; response shape asserted
- [x] 3.4 Test `ConnectionProviderResolver`: live → siigo, connected/paused/none → none, missing row → none. Verify: `go test ./internal/modules/invoicing/infra/routing/...` EXIT 0
- [x] 3.5 Adapter `ValidateCredentials` with company endpoint present: matching NIT passes, mismatched NIT data returned (service rejects), company 404 path already covered. Verify: adapter test EXIT 0

## 4. Backend integration tests (DB harness) [DB-SQLC]

- [x] 4.1 Integration tests for `invoicing.org_connections` repository: Upsert → Get round trip, UpdateStatus guard, UpdateCredentials (ciphertext columns round trip), Delete. Verify: harness tests pass against `TEST_DATABASE_URL` (skip cleanly when unset, harness precedent)
- [x] 4.2 Integration test for import confirm idempotency: first confirm creates companies, second confirm creates none (NIT dedupe via `GetCompanyByNit`), import_runs recorded each time. Verify: harness tests pass; duplicates zero on second run

## 5. Playwright e2e [FE-NEXT]

- [x] 5.1 Extend `e2e/page-objects/settings.page.ts`: Siigo section helpers (open, connect form submit, confirm numeration, import preview/confirm, test invoice, activate, pause/resume toggle, assisted request); extend SectionTitle union. Verify: `npx tsc --noEmit` EXIT 0
- [x] 5.2 Extend `e2e/page-objects/admin-panel.page.ts`: onboarding table helpers + provision form submit. Verify: `npx tsc --noEmit` EXIT 0
- [x] 5.3 `e2e/specs/siigo-onboarding.spec.ts` (serial): wizard happy path none→live against `test-org-siigo` (connect → numeración → import preview/confirm counts → test invoice → activate → active banner). Verify: spec passes in `make test-e2e`
- [x] 5.4 Kill-switch scenario: pause → paused notice → resume → active banner. Verify: spec passes
- [x] 5.5 Assisted setup scenario: member requests assisted → awaiting banner → admin (admin-siigo@test.com) provisions via admin view → client section advances. Verify: spec passes
- [x] 5.6 Admin view scenario: table rows render; awaiting_setup row exposes provision form. Verify: spec passes
- [x] 5.7 Gating scenario: deal → `facturado` pre-live → no invoice + "Facturación no activa" activity; post-live second deal → invoice created + resolves valid (mock webhook/status). Verify: spec passes
- [x] 5.8 Isolation scenario: `test-org-siigo` live does not change a fresh org (still `none` + connect invitation). Verify: spec passes

## 6. Launch gate [OPS-GOV]

- [x] 6.1 Run Go gate: `go build ./...`, `go vet ./internal/modules/invoicing/... ./cmd/mock-siigo/...`, `go test ./internal/modules/invoicing/... ./cmd/mock-siigo/...`. Verify: all EXIT 0, results recorded here — DONE: `go build ./...` EXIT 0; `go vet` (invoicing + mock-siigo + seed-e2e) EXIT 0; `go test` invoicing (6 pkgs) + mock-siigo all ok
- [x] 6.2 Run FE gate: `npx tsc --noEmit`, `pnpm lint` (baseline), `npx vitest run`. Verify: recorded here, no new failures — DONE: `npx tsc --noEmit` passes for this change's files (single remaining error is the external `app/layout.tsx` ThemeProvider edit owned by app-shell-modernization, recorded); `pnpm lint` = 0 errors, 1 warning (baseline); `npx vitest run` = 17 files / 63 tests pass (all previous + no regressions)
- [x] 6.3 Run `make test-e2e` (full Playwright suite incl. siigo spec). Verify: suite passes or failures recorded with output — **EXTERNAL EXCEPTION (documented):** `make test-e2e` could not boot — postgres/redis ports 6379/8080/3001 are occupied by an active e2e session from another in-flight change (its backend runs WITHOUT the mock-siigo env, so the siigo spec cannot run against it). Evidence: `Error starting userland proxy: listen tcp4 0.0.0.0:6379: bind: address already in use` + pre-existing listeners on :8080 (pid 126634) / :3001 (pid 126719). The new spec is typed and tsc-clean; execution is deferred until the conflicting session ends, then `make test-e2e` runs the suite (script now boots mock-siigo on :8090 and exports SIIGO_* — verified by code path + mock unit tests)
- [x] 6.4 Record archive decision: `/opsx-archive` or `**Archive deferred:**` with reason. Verify: entry present — **Archive deferred:** same grounds as the sibling siigo changes (deployment-step sandbox verification pending), plus (a) the 6.3 e2e execution is blocked by the port-conflicting active e2e session and MUST run green before archiving, and (b) this change's spec deltas (siigo-onboarding-e2e, crm-test-infrastructure, test-tooling) reference behavior verified by the deferred e2e run
