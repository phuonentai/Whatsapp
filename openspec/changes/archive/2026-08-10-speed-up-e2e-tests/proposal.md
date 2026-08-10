## Why

The Playwright E2E suite (61 tests across 12 spec files in `next_b2b_starter/e2e/`) runs fully serially with fixed sleep waits, making it painfully slow locally and in CI. The suite is spec-driven and expected to grow, so the slow runtime is a growing tax on every `make test-e2e` run and every CI push. Speed should come from test design (parallelism, web-first assertions), not from weakening coverage.

## What Changes

- Parallelize test execution: set `fullyParallel: true` and derive `workers` from available CPU cores locally (explicit count in CI)
- Remove `storageState` config (`./e2e/storage-state.json`) — the global setup never writes it, so it is dead configuration
- Eliminate all fixed `waitForTimeout` sleeps in specs and page objects, replacing them with web-first assertions and actionability-based auto-waiting
- Remove dead `waitForLoadState("networkidle")` in `login.page.ts` (unused `LoginPage` class)
- Raise per-test timeout from 10s to 15s to give parallel workers headroom and reduce CI retry-induced slowness on loaded runners
- Add a no-progress watchdog to `go-b2b-starter/scripts/run_e2e.sh`: if the Playwright runner emits no output for 180s (a hang — runner never exits, so `cleanup` never fires), kill it and rerun the suite once; real test failures do NOT trigger retry

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `crm-test-infrastructure`: Adds requirements that the Playwright suite SHALL run with parallel workers, that tests MUST NOT use fixed sleep waits (replacing them with web-first assertions), and that the `make test-e2e` runner MUST self-recover from hangs via a no-progress watchdog that kills and retries the suite once after 180s of runner silence.

## Impact

- **Code**: `next_b2b_starter/e2e/playwright.config.ts`, all spec files under `next_b2b_starter/e2e/specs/`, page objects under `next_b2b_starter/e2e/page-objects/`, unused `next_b2b_starter/e2e/page-objects/login.page.ts`, `go-b2b-starter/scripts/run_e2e.sh` (watchdog + retry)
- **No changes** to application code, backend, or test assertions
- **Dev workflow**: `make test-e2e` (and `pnpm test:e2e`) complete faster; wall-clock runtime recorded before/after; hanging runs self-recover instead of leaving the shell stuck
- **Non-Goals**: No change to what the tests assert. No flake-hunting rewrites beyond wait removal. No new test cases. No backend or frontend application code changes. No changes to mock-auth behavior — per-test `X-Test-Org-ID` headers make parallel execution safe (no shared auth state). No local credential storage is introduced or modified. No CI-side retry or job-level timeout (watchdog is local-only; CI keeps its existing `retries: 2` and runner timeouts).
