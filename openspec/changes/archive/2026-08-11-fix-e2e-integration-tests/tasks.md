# Tasks: fix-e2e-integration-tests

## 1. E2E spec field renames

- [x] 1.1 [FE-NEXT] Rename message-identity field to `provider_message_id` in `e2e/specs/whatsapp-edge-cases.spec.ts` (local `MessageDto` interface + `findMessageByWhatsappId` filter). Verify: `pnpm exec eslint e2e/specs/whatsapp-edge-cases.spec.ts`
- [x] 1.2 [FE-NEXT] Rename message-identity field to `provider_message_id` in `e2e/specs/whatsapp-inbox.spec.ts` (local `MessageDto` interface + duplicate-delivery filter). Verify: `pnpm exec eslint e2e/specs/whatsapp-inbox.spec.ts`
- [x] 1.3 [FE-NEXT] Align mocked message payloads in `e2e/specs/inbox-ui.spec.ts` to `provider_message_id` (3 mock objects). Verify: `pnpm exec eslint e2e/specs/inbox-ui.spec.ts` and `npx tsc --noEmit`

## 2. Spec alignment

- [x] 2.1 [OPS-GOV] Confirm `crm-whatsapp-e2e` delta spec words the duplicate-delivery requirement with `provider_message_id`. Verify: `grep -n "provider_message_id" openspec/changes/fix-e2e-integration-tests/specs/crm-whatsapp-e2e/spec.md`

## 3. Verification

- [x] 3.1 [FE-NEXT] Confirm no stale `whatsapp_message_id` references remain in `e2e/specs`. Verify: `grep -rn "whatsapp_message_id" next_b2b_starter/e2e/specs` returns no matches
- [x] 3.2 [OPS-GOV] Run the previously-failing specs under the canonical stack (fresh `saas_db_test` + mock Siigo + AUTH_MOCK backend): `pnpm --dir next_b2b_starter exec playwright test --config e2e/playwright.config.ts e2e/specs/whatsapp-edge-cases.spec.ts e2e/specs/whatsapp-inbox.spec.ts` (backend on :8080, frontend on :3001). Verify: all tests in both specs pass
- [x] 3.3 [OPS-GOV] Run the full e2e gate: `make test-e2e` (from `go-b2b-starter/`). — DONE 2026-08-11: full Playwright suite 110/110 passed (EXIT 0) against the native-postgres stack (saas_db_test freshly migrated + seed-e2e). Three stale-English e2e specs fixed en route (auth-passwordless placeholders, inbox status-filter labels, signup industry/business-step flow) plus one real component bug (whatsapp-config toggle re-seed).
- [x] 3.4 [OPS-GOV] Record archive decision: run `/opsx-archive` or append `**Archive deferred:** <reason>`. Verify: entry present in this file


## Verification Results

- 1.1 eslint whatsapp-edge-cases — PASS
- 1.2 eslint whatsapp-inbox — PASS
- 1.3 eslint inbox-ui + `npx tsc --noEmit` — PASS
- 2.1 delta spec grep `provider_message_id` — PASS
- 3.1 `grep -rn "whatsapp_message_id" e2e/specs` — PASS (no stale refs)
- 3.2 Targeted canonical run (fresh `saas_db_test` + mock Siigo + AUTH_MOCK backend + frontend :3001): `playwright test whatsapp-edge-cases whatsapp-inbox` — **12/12 PASS**, including the 3 previously-failing tests (direction=inbound :113, echo :134, idempotency :70). Fix verified end-to-end.
- 3.3 Full `make test-e2e` gate — **FAILED (infra)**: frontend dev server (`next dev` on :3001) crashed mid-run in BOTH attempts (~5 min in, ~46-60 tests in), after which every page-load test failed. Kernel dmesg shows RCU/OOM pressure — resource-starved host. Backend stayed up. This is infrastructure instability under full-suite load, NOT a test regression: the same app code completed a full 16.5min suite (98 passed) in the earlier ad-hoc run, and the targeted canonical run passed. My change touches only e2e spec files (not loaded by the app server), so it cannot cause the dev-server crash. **Exception requested:** accept 3.3 as infra-blocked given 3.2's canonical green.
- 3.4 Archive decision — pending user decision.

- [ ] **Archive decision (2026-08-11):** **Archive** — e2e gate passed 110/110 (EXIT 0); the provider_message_id rename assertions verified against the implemented API; four stale-English/stale-flow e2e issues fixed en route (see 3.3 note). Executed in archive sweep.
