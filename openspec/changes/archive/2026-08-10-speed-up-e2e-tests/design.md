## Context

The Playwright suite in `next_b2b_starter/e2e/` has 61 tests across 12 spec files. `playwright.config.ts` sets `fullyParallel: false` and `workers: 1`, so all 61 tests execute serially. Tests also contain ~15 fixed `waitForTimeout(300–500)` sleeps, plus a `waitForLoadState("networkidle")` in the unused `LoginPage` class. Authentication is mock-based: each test sets its own `X-Test-Org-ID` header/cookie via `setExtraHTTPHeaders` or the `loginAs` fixture. There is no shared session file, so there is no cross-test shared state.

Two neighboring active changes touch the same area: `add-crm-e2e-tests` (created the suite, in-progress) and `add-ci-pipeline` (wired `make test-e2e` and the CI e2e job). This change is additive and orthogonal to both — it does not change assertions, only execution strategy and wait mechanics.

## Goals / Non-Goals

**Goals:**
- Run the 61 tests in parallel across multiple workers (locally: auto-derived from CPU cores; CI: explicit count)
- Eliminate every fixed `waitForTimeout` and `networkidle` wait in specs and page objects, replacing them with Playwright web-first assertions and actionability auto-waiting
- Remove dead `storageState` config that `global-setup.ts` never produces
- Reduce suite wall-clock time meaningfully, measured before/after
- Ensure `make test-e2e` self-recovers when the Playwright runner hangs (no output) instead of leaving the shell stuck forever

**Non-Goals:**
- No changes to test assertions or covered behavior
- No backend or frontend application code changes
- No new test cases, no visual regression tests
- No change to the mock-auth flow — parallel execution relies on the existing per-test header isolation
- No CI-side retry, job-level timeout, or `timeout-minutes` changes — the watchdog is local-only (`run_e2e.sh`); CI keeps its existing `retries: 2`

## Decisions

| Decision | Choice | Alternatives Considered | Rationale |
|----------|--------|------------------------|-----------|
| Parallelism config | `fullyParallel: true`; `workers: process.env.CI ? 4 : undefined` | Fixed `workers: 4` everywhere; keep serial | Parallel is safe because each test authenticates independently via its own `X-Test-Org-ID` header — no shared storage-state file, no cross-test DB collisions (all specs use `Date.now()`-unique records). `undefined` locally lets Playwright pick CPU count; explicit 4 in CI keeps runner resource use bounded. |
| `storageState` | Remove `use.storageState` | Generate the file in `global-setup.ts` | The file is never written today; tests never rely on it. Removing the dead config is honest; generating it would add fake machinery. Mock headers remain the sole auth mechanism. |
| Hard waits | Replace with web-first assertions (`expect(...).toBeVisible()`, `toBeEnabled()`, row-count asserts) | Keep waits but shorten | Fixed sleeps are wrong even at 100ms — they pass late and fail early. Playwright actionability waiting already covers clicks/fills; only state that is not the action target needs an explicit `expect` await. |
| Dead code | Delete `LoginPage`/`login.page.ts` and the `networkidle` wait | Leave in place | Unreferenced by any spec; its `networkidle` wait is a known hang risk for apps with polling/WebSockets. Removing it prevents accidental future use. |
| Timeouts | `timeout: 15_000`, keep `expect: { timeout: 10_000 }` | Leave at 10s | Parallel workers share CPU; raising the per-test budget reduces spurious timeouts on loaded machines, which in CI trigger `retries: 2` and multiply wall-clock cost. |
| Hang recovery | No-progress watchdog in `run_e2e.sh` (kill + rerun once after 180s of runner silence) | Total-run budget via `timeout`; rely on Playwright `retries` | A total-run budget kills healthy slow runs (81 tests, parallel, loaded machines exceed 3 min legitimately). Playwright `retries` only retries a *failed test*, not a hung runner — a hang means the runner never exits, so `cleanup` never fires. No-progress detection (log-file mtime) distinguishes a healthy slow run (output flowing) from a hang (silent). Runs under a PTY via `script -q -c` so Playwright line-buffers instead of block-buffering its reporter output. |

### Wait replacement strategy

`waitForSelector` and `waitForLoadState("load")` that wait on a specific condition are acceptable to convert opportunistically but are NOT the primary target — they are conditional, not fixed-duration. Only fixed-duration waits (`waitForTimeout`, `networkidle`) MUST be removed. The general pattern:

```
await actionThatTriggersAsyncUi();          // click, fill, submit
await expect(page.locator(selector)).toBeVisible();  // resolves the moment it appears
```

For data-table assertions (e.g., row appears after create), assert on the row locator itself:

```
await expect(page.locator(`tr:has-text("${phone}")`)).toBeVisible();
await expect(page.locator("tbody tr")).toHaveCount(expected);
```

For the two "delete confirms removal" tests, `toBeHidden`/`toHaveCount(0)` replaces the sleep-then-count check.

## Risks / Trade-offs

- [Parallel workers share one backend + DB] → All specs create uniquely-named records (`Date.now()`-suffixed), and `X-Test-Org-ID` scopes each test to its org's tenant. Seed data is read-only (orgs, plans, RBAC accounts). Low risk; validated by a full suite run after the change.
- [Assertion must be chosen carefully so it is not vacuous] → Each replaced wait lands on the specific UI effect of the preceding action (row visibility, error text, detail view), not a generic page-level wait. Verified by reading each test in context during the edit.
- [Parallel run unmasks latent coupling between specs] → If a spec depended on serial ordering (unlikely — all specs are self-contained), the parallel run will fail fast and that spec gets fixed, not re-serialized. This is the desired outcome of this change.
- [Timeout raise masks genuinely slow tests] → Acceptable trade-off; the suite was created recently and its timing is dominated by serial execution, not individual slow tests. Re-audit after the change if any single test exceeds 15s.
- [No-progress watchdog false-fires on a healthy but block-buffered run] → Mitigated by running Playwright under a PTY (`script -q -c`) so the `list` reporter flushes per test; the watchdog watches log-file mtime, so any emitted output resets the silent clock. A 180s silence threshold is far beyond the longest intra-test quiet period.
- [Watchdog kills the runner but stray chromium/worker processes linger] → The retry path runs `pkill -f "chromium|playwright"` before rerunning, clearing leaked browser/worker processes that would otherwise block ports or file locks.
- [Retry replays the same hang infinitely] → Retry is bounded: at most 2 attempts (1 initial + 1 retry). A persistently hung env fails instead of looping.

## Migration Plan

1. Update `playwright.config.ts`: `fullyParallel: true`, `workers` expression, remove `storageState`, raise `timeout`.
2. Remove dead `login.page.ts`; grep to confirm no spec imports it.
3. Replace fixed waits spec-by-spec and page-object-by-page-object.
4. Add the no-progress watchdog + retry loop to `go-b2b-starter/scripts/run_e2e.sh` (see below).
5. Verify: `make test-e2e` (or `pnpm test:e2e`) green; record before/after wall-clock.
6. Rollback: revert config + wait edits + script edit via git — the change is confined to `next_b2b_starter/e2e/` and `go-b2b-starter/scripts/run_e2e.sh`, no schema or auth impact. No Stytch/tenant state involved.

### Watchdog sketch (for `run_e2e.sh`)

```
run_attempt() {
  local logfile="/tmp/e2e-attempt-$1.log"
  script -q -c "pnpm --dir \"$FRONTEND_DIR\" test:e2e" "$logfile" &
  local pid=$!
  local silent_since
  while kill -0 "$pid" 2>/dev/null; do
    sleep 5
    local mtime
    mtime=$(stat -c %Y "$logfile")
    if [ "$(( $(date +%s) - mtime ))" -gt 180 ]; then
      echo "!! no progress for 180s — killing attempt $1"
      kill -TERM "$pid"; wait "$pid" 2>/dev/null
      pkill -f "chromium|playwright" 2>/dev/null || true
      return 124
    fi
  done
  wait "$pid"
  return $?
}

run_attempt 1 || status=$?
# retry only on hang (124); real failures (1) exit as-is
```
