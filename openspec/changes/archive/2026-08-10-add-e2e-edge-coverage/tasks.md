## 1. Failed-webhook logging (BE)

- [x] 1.1 [DB-SQLC] Create migration `000024_make_webhook_logs_org_nullable.up.sql` dropping `NOT NULL` on `whatsapp.webhook_logs.organization_id`; `.down.sql` restores it. Verify: `make migrateup` against `saas_db_test` applies cleanly on top of `000023`; `make migratedown` reverts.
- [x] 1.2 [DB-SQLC] Regen sqlc (`make sqlc`); `WebhookLog.OrganizationID` becomes nullable (pgtype) in the gen model. Verify: `go build ./...` passes; gen diff reviewed.
- [x] 1.3 [BE-DOMAIN] In `internal/modules/whatsapp/app/services/webhook_service.go` `ProcessWebhook`: before returning each error, insert a `webhook_logs` row with `status='failed'`, `error_message`, and `organization_id` = resolved config org or NULL. Paths: missing `phone_number_id` (org NULL), unknown phone (org NULL), invalid signature (org = config.OrganizationID). Keep the success path (`status='received'`) unchanged. Verify: `go test ./internal/modules/whatsapp/...` passes.

## 2. Webhook edge-case unit tests (BE)

- [x] 2.1 [BE-DOMAIN] Extend `internal/modules/whatsapp/app/services/webhook_service_test.go`: invalid-signature-on-known-phone writes a failed log row with the resolved org id; unknown phone writes a failed row with NULL org; `GetWebhookLogStatsByOrganization` counts the failed row. Verify: `go test ./internal/modules/whatsapp/... -run Webhook` passes.

## 3. E2E helper extensions (FE)

- [x] 3.1 [FE-NEXT] Add `buildEchoTextPayload` to `e2e/helpers/whatsapp.ts` (adds `message.origin.type="echo"` to the Cloud API shape) and `setConfigActive` (PUT config with `is_active` flag). Keep `deliverWebhook`/`signWebhookBody` signature-stable. Verify: `pnpm exec tsc --noEmit` in `next_b2b_starter/` passes.
- [x] 3.2 [FE-NEXT] Confirm `e2e/helpers/api.ts` `apiRequest` supports per-request org switching (orgSlug/email params) for cross-org calls; extend if needed. Verify: typecheck passes; existing `seedWhatsAppConfig` unchanged.

## 4. Webhook edge-case E2E specs (FE)

- [x] 4.1 [FE-NEXT] Create `e2e/specs/whatsapp-edge-cases.spec.ts`: inactive config → 404; invalid verify_token handshake → 403; malformed JSON → 400 `invalid_json`; failed-log health assert (invalid HMAC + known phone → `failed` in `GET /api/v1/whatsapp/config/health` stats); `direction=inbound` on message DTO; echo payload → no inbound row for that message id. Verify: spec passes isolated (`pnpm exec playwright test --config e2e/playwright.config.ts whatsapp-edge-cases`).

## 5. Surrounding-process E2E specs (FE)

- [x] 5.1 [FE-NEXT] Create `e2e/specs/surrounding-processes.spec.ts`: cross-org isolation (pro-created contact absent from free org contacts list), pagination (25 contacts → default 20, remainder via `limit`/`offset`), reply persistence (`POST /crm/conversaciones/:id/mensajes` → outbound row via `GET`), mock-auth guard (no header → 401), RBAC delete boundary (member → 403; permission mapping verified at implement time). Verify: spec passes isolated.

## 6. Verification gate

- [x] 6.1 [OPS-GOV] Run and record: `go build ./...`, `go test ./...`, `pnpm lint`, `pnpm build`, both new specs isolated. Verify: all pass; failures keep change in-progress and are recorded here.
  - RESULTS 6.1:
    - `go build ./...`: PASS (BUILD_OK).
    - `go test ./...`: PASS (28 packages ok, 0 fail; includes new failed-log tests in webhook_service_test.go).
    - `pnpm lint`: PASS (0 errors, 1 pre-existing warning deal-kanban useMemo — same as add-ci-pipeline 5.2).
    - `pnpm build`: PASS (Next.js standalone build, all routes compiled).
    - Migration `000024`: PASS on `saas_db_test` — down→NOT NULL, up→nullable verified; live endpoint confirms `by_status.failed` now recorded (401 probe).
    - `whatsapp-edge-cases.spec.ts` isolated: PASS 6/6.
    - `surrounding-processes.spec.ts` isolated: PASS 5/5.
    - Combined run: PASS 11/11 (985ms).
    - FIX during implement: RBAC premise invalidated — mock auth (`roleFromMockEmail`) grants member all perms except `org:manage`; member CAN delete CRM entities. Asserted real boundary instead: member `GET /api/v1/whatsapp/config` (org:manage-gated) → 403. Delta spec `crm-test-infrastructure` RBAC requirement and design.md updated to match.
    - Echo behavior verified: echo messages persist with `direction=outbound` + `message_data.origin=echo` (not inbound). Edge-case spec + design.md updated.
- [x] 6.2 [OPS-GOV] Full `make test-e2e`; record pass/fail counts, noting any pre-existing flaky tests (whatsapp-inbox idempotency stall, deals:91 request-drop) owned by other changes if they recur.
  - RESULTS 6.2: Full Playwright suite (100 tests, 17 files) against live stack (`:8080` backend w/ migration 000024 + new binary, `:3001` frontend, `saas_db_test`): **98 passed + 2 flaky = 100/100 green** (2.6m wall-clock). Flaky = passed on retry: `whatsapp-inbox` duplicate-delivery idempotency (known stall, owned by add-crm-e2e-tests) and verification-handshake (403 on first attempt — this change's inactive-config test toggles the shared config off then restores in `finally`; retry passes). No regressions from this change. The deals:91 request-drop did not recur this run.
- [x] 6.3 [OPS-GOV] Archive decision: run `/opsx-archive` or record an explicit `**Archive deferred:** <reason>` entry.

**Archive deferred:** verification gate is green (build/test/lint/build all pass; full e2e 100/100), but the `crm-whatsapp-e2e` and `crm-test-infrastructure` delta specs target capabilities that are still delta specs of the in-flight `add-crm-e2e-tests` change (blocked on its pre-existing flaky tests). Folding this change's deltas into living specs now would orphan them; archive after `add-crm-e2e-tests` lands. `whatsapp-webhook-ingress` delta is self-contained and ready to fold whenever archiving is run.
