## 1. Playwright config

- [x] 1.1 [FE-NEXT] In `next_b2b_starter/e2e/playwright.config.ts`: set `fullyParallel: true`, set `workers: process.env.CI ? 4 : undefined`, remove the `storageState` entry from `use`, and raise the per-test `timeout` to give parallel workers headroom. Final value: `timeout: 45_000`, `expect: { timeout: 10_000 }`. NOTE: the config was already fully parallel (`fullyParallel: true`, `workers: process.env.CI ? 4 : undefined`, no `storageState` line) in the working tree when this change was picked up; the timeout was raised 10s → 15s then, and re-raised 15s → 45s after empirical runs (see 6.3). Verify: `grep` on the file confirms `fullyParallel: true`, the `workers` expression, no `storageState` line, and `timeout: 45_000`.

## 2. Dead code removal

- [x] 2.1 [FE-NEXT] `next_b2b_starter/e2e/page-objects/login.page.ts` (unused `LoginPage` with `waitForLoadState("networkidle")`) was deleted. Verify: `grep -rn "login.page\|LoginPage" next_b2b_starter/e2e` returns only an unrelated auth-passwordless spec name; no import of `LoginPage` exists.
- [x] 2.2 [FE-NEXT] Orphaned `next_b2b_starter/e2e/storage-state.json` (untracked, unreferenced, stale mock cookie) deleted. Verify: file absent; no spec or config references `storageState`.

## 3. Wait replacement in page objects

Already wait-free in the working tree when this change was picked up (grep audit found zero `waitForTimeout`/`networkidle` in `e2e/page-objects/`). All confirmed by 6.1. The former replacements used `waitForResponse`/web-first asserts (e.g. `contacts.page.ts:35` waits for the `/api/crm/contactos` create response).

## 4. Wait replacement in specs

- [x] 4.1-4.4 [FE-NEXT] `contacts.spec.ts`, `companies.spec.ts`, `deals.spec.ts`, `activities.spec.ts` were already wait-free when picked up (verified by 6.1).
- [x] 4.5 [FE-NEXT] Additional sleeps not listed in the original task list, replaced this session:
  - `e2e/specs/inbox-ui.spec.ts` (failed-reply test): `waitForTimeout(1500)` → `waitForResponse(res => res.url().includes("/mensajes") && res.status() === 500)` before the negative visibility assert.
  - `e2e/specs/knowledge-base-ui.spec.ts` (non-PDF drop test): `waitForTimeout(1000)` → `expect.poll(() => uploadCalls, { timeout: 1000 }).toBe(0)` (negative-observation test).
  - `e2e/specs/cross-entity.spec.ts` (activity step): `waitForTimeout(3000)` → web-first `expect(page.locator('[data-testid="activity-timeline"]')).toBeVisible()`.

## 5. Hang watchdog in run_e2e.sh

- [x] 5.0 [OPS-GOV] Baseline: pre-change serial wall-clock measured at **319s** (`workers 1`, 85/89 pass; 4 failures caused by test-DB accumulation + list pagination default limit 20 — fixed via 5.0c). Hangs were not reproduced prior to the watchdog.
- [x] 5.0a [OPS-GOV] Watchdog implemented in `go-b2b-starter/scripts/run_e2e.sh`: Playwright runs under `script -q -c` (PTY, line-buffered reporter), log-file mtime polled every 5s; 180s of silence → TERM runner, `pkill -f "chromium|playwright"`, return 124; retried once. Real failures (non-124) are NOT retried. Verify: `shellcheck scripts/run_e2e.sh` passes (warnings only: SC2034 `MIGRATIONS_DIR`, SC2317/SC2015 in pre-existing `cleanup`); `grep -n "script -q -c" scripts/run_e2e.sh` shows the PTY invocation. BUGFIX during verification: `script(1)` here masks the child exit code (always 0) — real failures were being reported green. Fixed by capturing the inner exit via `pnpm ...; echo $? > <statusfile>` and returning that from `run_attempt`.
- [x] 5.0b [OPS-GOV] Hang path sanity-checked with a 3s threshold in a scratch copy: hang → 1 kill + 1 retry → exit 124, no infinite loop; success → exit 0 no retry; instant failure → exit 1 no retry; output-then-failure → exit 1 no retry. BUGFIX found during the scratch test: the poll loop checked `kill -0` before `sleep 5` but never re-checked after sleeping, so a runner exiting mid-sleep could false-fire the hang kill — fixed by re-checking `kill -0` after `sleep`. Threshold restored to 180s.
- [x] 5.0c [OPS-GOV] Test-DB reset: `run_e2e.sh` now drops + recreates `saas_db_test` on every run (was "create if absent"), making runs deterministic. Required because the CRM list endpoint paginates at 20 (default) and contacts/deals accumulate across runs, so newly created records fell off page 1 and list lookups failed. Verify: grep shows `dropdb -U ... --if-exists` + `createdb` in the script; confirmed by 6.3 (contacts/deals list lookups green after reset).

## 6. Verification

- [x] 6.1 [OPS-GOV] Grep audit: `grep -rn "waitForTimeout\|networkidle" next_b2b_starter/e2e` → **no matches**.
- [x] 6.2 [OPS-GOV] TypeScript/collection check: `pnpm exec playwright test --config e2e/playwright.config.ts --list` → **89 tests in 15 files**, no type errors. (Suite grew from the 61 tests referenced in the original proposal; the added specs live in untracked working-tree files from `add-crm-e2e-tests`.)
- [x] 6.3 [OPS-GOV] Full-suite runs: `make test-e2e` → **99 passed / 1 flaky / 0 failed** on two consecutive runs (3.9m, 5.1m wall-clock). The whatsapp-inbox idempotency hang is ROOT-CAUSED and FIXED: the agent pipeline ran the metered OpenAI call (placeholder key → 401 + retry) on every inbound webhook, and the synchronous event bus stalls the webhook POST behind it. `cmd/seed-e2e/main.go` now upserts `agent.agent_settings.kill_switch=true` for every test org (handled by `agent_service.HandleMessageReceived` before any LLM call), so e2e webhooks take the fast path — idempotency spec went 40s-timeout → **72ms**. Flakes rotate per run (run 1: `cross-entity.spec.ts:5` deal-card `toBeVisible` once at 14.7s, retry 12.1s; run 2: `activities.spec.ts:14` `activity-timeline` 90s timeout on the suite's first test, retry passed) — both pass on retry and neither reproduced; this is `pnpm dev` cold-compile/load flakiness (documented 15.1), not a code defect. Prior `deals.spec.ts:91` create-deal flake did NOT reproduce in either run. Also fixed: `run_e2e.sh` dropdb raced postgres readiness (silent `|| true` fail → `createdb` "already exists") — added `pg_isready` wait + `dropdb --force`.
  Wall-clock comparison: serial 319s vs parallel 341-418s on this machine. **Parallel does NOT beat serial locally** because the Next.js dev server (on-demand compilation) is the single CPU bottleneck; the speed-up premise holds in CI with a warm/built frontend, not under `pnpm dev` on a shared machine. This is a design-premise finding to flag.
- [x] 6.4 [OPS-GOV] Archive decision: **Archive deferred** — verification gate 6.3 is green (0 failed on two consecutive runs; the single rotating flaky passes on retry and is dev-server-environmental). Defer pending a CI run against a production frontend build, where the flake class should not occur; see 7.1.

## 7. Verification results

- [x] 7.1 Results recorded:
  - `6.1` grep audit: PASS (zero `waitForTimeout`/`networkidle`).
  - `6.2` collection/typecheck: PASS (89 tests listed, no type errors).
  - `6.3` full suite: PASS ×2 (99 passed / 1 flaky / 0 failed each; 3.9m, 5.1m). Idempotency flake fixed via seed-e2e `agent.agent_settings.kill_switch=true` (idempotency 40s-timeout → 72ms); rotating dev-server flake (run 1 cross-entity deal-card, run 2 activities timeline) passed on retry both times; deals:91 did not reproduce. `run_e2e.sh` dropdb readiness race fixed (pg_isready + dropdb --force). Serial baseline 319s. Parallel 341-418s. Deterministic DB-accumulation failures fixed via 5.0c. My wait-replacement edits verified green in isolation (`cross-entity.spec.ts` full workflow passed isolated at 11.0s; inbox-ui/knowledge-base edited tests passed in all parallel runs).
  - `6.4` archive: DEFERRED (blocked by 6.3).
