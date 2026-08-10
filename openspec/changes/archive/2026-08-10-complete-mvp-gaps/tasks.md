## 1. Provider-aware billing UI [FE-NEXT]

- [x] 1.1 `app/dashboard/settings/components/subscription-tab.tsx`: branch cancellation by `isMercadoPagoEnabled()` — MP enabled → `cancelMPSubscription` (existing action → `POST /api/subscriptions/mp-cancel`), else existing Polar `cancelSubscription`; mirror the existing cancel-dialog UX; replace the "Billed via Polar" fallback (line 197) with provider-aware copy. Verify: `npx tsc --noEmit`; `pnpm build`; `pnpm lint` at documented baseline (no new violations)
- [x] 1.2 `components/billing/subscription-paywall.tsx`: guard inactive-state copy by `isMercadoPagoEnabled()` — MP copy references PSE/Nequi checkout, Polar copy unchanged. Verify: `npx tsc --noEmit`; `pnpm build`; `pnpm lint` at documented baseline (no new violations)

## 2. Task-state reconciliation [OPS-GOV]

- [x] 2.1 `add-mercadopago-billing/tasks.md`: mark 3.2/3.3/3.4 done with note — SQLC queries verified present (`internal/db/postgres/sqlc/gen/organizations.sql.go:372,633`; `sqlc/query/organizations.sql:55,60`). Verify: `grep` shows `- [x]` on those tasks + note
- [x] 2.2 `add-crm-e2e-tests/tasks.md`: mark 2.3 done (verified: `test:e2e` in `next_b2b_starter/package.json:9`) and 11.1 done (verified: `run-frontend` + `run-e2e` jobs in `go-b2b-starter/.gitlab-ci.yml:22,41`). Verify: `grep` shows `- [x]` + notes — **RESOLVED VIA ARCHIVE:** the owning change was archived as `openspec/changes/archive/2026-08-10-add-crm-e2e-tests` with both tasks already `- [x]` and verification notes dated 2026-08-10 (2.3: `test:e2e` present package.json:9; 11.1: `run-e2e` stage present .gitlab-ci.yml:41, fresh migrate verified). Archived tasks.md is not editable per governance; nothing further to reconcile
- [x] 2.3 Record archive-deferral entries: `fix-migration-renumber/tasks.md` task 3.4 and `add-whatsapp-embedded-signup/tasks.md` task 7.3 (`**Archive deferred:** <reason>` per governance-workflow spec). Verify: entries present in both files

## 3. E2E flaky-spec hardening [FE-NEXT]

- [x] 3.1 `e2e/specs/whatsapp-inbox.spec.ts:65` (duplicate-delivery idempotency): replace bare deliver `fetch` with retry/`expect.poll` so an intermittently dropped request does not stall the test ≥45s. Verify: `npx tsc --noEmit` on e2e config; full-suite run deferred (environment — port 3001 occupied; recorded in this file) — DONE 2026-08-10: `deliverWebhook` (`e2e/helpers/whatsapp.ts`) sends `signal: AbortSignal.timeout(10_000)` (stall → fast AbortError), test retries delivery up to 4× per send (no `waitForTimeout`, preserves speed-up-e2e-tests 6.1 audit). `tsc --noEmit` + eslint clean
- [x] 3.2 `e2e/specs/deals.spec.ts:91` (create deal linked to contact/company): make `waitForResponse` on `POST /api/crm/negocios` resilient to Next dev-server request drops (retry POST or retry-wait window) per design D4. Verify: `npx tsc --noEmit` on e2e config; full-suite run deferred (environment) — DONE 2026-08-10: replaced `waitForResponse` with `expect.poll` against `/crm/negocios` (15s timeout), matching the existing company/contact poll pattern in the same test. `tsc --noEmit` + eslint clean

## 4. Verification gate [OPS-GOV]

- [x] 4.1 Run gate and record results here: `npx tsc --noEmit`, `pnpm build`, `pnpm lint` (baseline: 13 errors + 1 warning documented in `fix-frontend-eslint-flat-config` 3.1 — no new violations), `go build ./...`, `go test ./...` (backend untouched, sanity). Verify: all recorded commands pass; failures keep change in-progress with notes — DONE 2026-08-10:
  - `npx tsc --noEmit` (next_b2b_starter) EXIT 0
  - `pnpm build` (next_b2b_starter) EXIT 0
  - `pnpm lint` EXIT 0 — 0 errors, 1 warning (`components/crm/deal-kanban.tsx:172` `react-hooks/exhaustive-deps`, pre-existing file untouched by this change); below documented baseline
  - `go build ./...` (go-b2b-starter, PATH+=/usr/local/go/bin) EXIT 0
  - `go test ./...` EXIT 0 (all packages pass)
  - E2E full suite: **deferred (environment)** — port 3001 occupied by respawning `next-server`; flaky-spec fixes typechecked + linted (3.1/3.2); execution via CI/deployment
- [x] 4.2 Record archive decision: run `/opsx-archive` or append `**Archive deferred:** <reason>` here. Verify: entry present per governance-workflow spec — **Archive deferred:** change complete; gate passed for all runnable commands (FE tsc/build/lint, BE build/test). Archiving deferred until (a) the full Playwright suite runs green in CI against the fixes in 3.1/3.2 (environment-blocked here: port 3001), and (b) MP-cancel UX verified against a deployed env with `NEXT_PUBLIC_MERCADOPAGO_PLAN_ID` set (live-credential deferral, consistent with wire-mercadopago-billing 13.x). Delta spec `billing-provider-ux` is stable and ready to fold in on `/opsx-archive`.
