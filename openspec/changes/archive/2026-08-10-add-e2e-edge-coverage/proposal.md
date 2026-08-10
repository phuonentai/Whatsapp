## Why

The E2E suite (15 specs / 89 tests) covers happy-path CRUD and primary webhook error cases, but several spec-mandated edge cases and surrounding processes are untested. The living `whatsapp-webhook-ingress` spec requires failed webhooks to be logged with `status='failed'`, yet `webhook_service.go` never writes a failed row and `whatsapp.webhook_logs.organization_id` is `NOT NULL`, so unknown-phone failures cannot be logged at all. Surrounding-process behaviors — cross-org data isolation, pagination limits, outbound reply persistence, the mock-auth guard, and the RBAC delete boundary — have no coverage. Regressions in these paths reach production undetected.

## What Changes

- Add BE failed-webhook logging: migration making `whatsapp.webhook_logs.organization_id` nullable (new migration `000024`), and `webhook_service.go` writing a `status='failed'` row with `error_message` before returning signature/unknown-phone errors. Org-resolvable failures carry `organization_id`; unresolvable ones log NULL.
- Extend `e2e/helpers/whatsapp.ts` with payload builders for edge cases (echo origin, inactive config toggle) and `e2e/helpers/api.ts` with per-request org switching for cross-org calls.
- Add E2E specs/assertions for webhook edge cases: inactive config → 404, invalid `verify_token` handshake → 403, malformed JSON → 400 `invalid_json`, failed webhook logged (health stats show `failed`), `direction=inbound` asserted on the message DTO, echo messages not rendered as inbound.
- Add surrounding-process E2E: cross-org data isolation, pagination (default 20, offset), outbound reply persistence via API, mock-auth guard (missing `X-Test-Org-ID` → 401), RBAC delete boundary (member → 403).
- Add Go unit tests for failed-webhook logging paths.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `whatsapp-webhook-ingress`: failed webhooks SHALL be logged with `status='failed'`; `organization_id` SHALL be nullable for unresolvable orgs.
- `crm-whatsapp-e2e`: adds edge-case scenarios (inactive config, invalid token, malformed payload, failed-log assertion, direction=inbound, echo).
- `crm-test-infrastructure`: adds surrounding-process test requirements (cross-org isolation, pagination, reply persistence, mock-auth guard, RBAC delete).

## Impact

- **Code (BE)**: `000024_make_webhook_logs_org_nullable.up/down.sql`, `webhook_service.go`, sqlc gen models (`WebhookLog.OrganizationID` → nullable), webhook unit tests.
- **Code (FE)**: `e2e/helpers/whatsapp.ts`, `e2e/helpers/api.ts` (org switch), new `e2e/specs/whatsapp-edge-cases.spec.ts` + `e2e/specs/surrounding-processes.spec.ts`, edits to `whatsapp-inbox.spec.ts`.
- **No UI / no app runtime behavior change** beyond failed-logging compliance.
- **DB**: one migration (nullable org_id). Rollback: `.down.sql` restores `NOT NULL`.
- **Dev workflow**: new isolated Playwright spec files run alongside the existing suite; `make test-e2e` total count grows.

## Non-Goals

- No change to inbound message processing, echo/event publishing semantics, or the 401/404 HTTP mappings (already spec-compliant).
- No new RBAC roles or permission changes.
- No outbound WhatsApp sends (requires live token) — reply persistence asserted via the offline mock-id path.
- No visual regression / no new browser targets.
- No pagination API changes — behavior asserted as-is.
- No local credential storage is introduced or modified.

## Assumptions

- The e2e capabilities currently living as delta specs of the in-flight `add-crm-e2e-tests` change will be folded into living specs on that change's archive; this change's deltas are additive and merge cleanly.
- `make test-e2e` full-suite verification may still hit the two pre-existing flaky tests (whatsapp-inbox idempotency stall, deals:91 request-drop); the verification gate records them as owned by other changes.
