# Change Proposal: fix-e2e-integration-tests

## Why

Three WhatsApp e2e tests fail at HEAD: the backend renamed the persisted message-key field from `whatsapp_message_id` to `provider_message_id` (committed in `internal/db/postgres/sqlc/query/crm.sql`, generated sqlc models, `crm/domain/message.go` JSON tag, and the frontend `MessageDto`), but the e2e specs still filter by the old `whatsapp_message_id`. Lookups therefore never match, so `whatsapp-edge-cases.spec.ts` (direction=inbound, echo persistence) and `whatsapp-inbox.spec.ts` (duplicate-delivery idempotency) fail deterministically in every environment. The living `crm-whatsapp-e2e` spec repeats the stale field name.

## What Changes

- Rename the message-identity field used by e2e assertions from `whatsapp_message_id` to `provider_message_id` in `e2e/specs/whatsapp-edge-cases.spec.ts` and `e2e/specs/whatsapp-inbox.spec.ts` (local `MessageDto` interface + filter expressions), matching the backend API contract.
- Align the mocked message payloads in `e2e/specs/inbox-ui.spec.ts` with the real `MessageDto` shape (`whatsapp_message_id` → `provider_message_id`) for consistency; not a failure driver.
- Update the `crm-whatsapp-e2e` living spec wording so the source of truth names the implemented field.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `crm-whatsapp-e2e`: the duplicate-delivery requirement SHALL identify persisted messages by `provider_message_id` (matching the backend API) instead of the renamed-away `whatsapp_message_id`.

## Impact

- **e2e specs only** (`next_b2b_starter/e2e/specs/whatsapp-edge-cases.spec.ts`, `whatsapp-inbox.spec.ts`, `inbox-ui.spec.ts`) and the `crm-whatsapp-e2e` spec wording.
- No backend code, frontend application code, API, or schema changes.
- Verification runs under the canonical `make test-e2e` stack (fresh `saas_db_test`, mock Siigo, `AUTH_MOCK_ENABLED`).

## Non-Goals

- No local credential, password, MFA, or session-token storage — Stytch B2B remains the sole identity/session authority; no storage change is introduced by this change.
- No backend or `agent`/`campaign` DTO changes; `whatsapp_message_id` in those DTOs is a separate contract, out of scope.
- No changes to the Siigo onboarding e2e spec or the flaky deals/tags tests (environment/DB-state dependent; owned by their respective changes).

## Rollback

- Git state: revert the change's commits (e2e field renames + spec wording); additive, no migrations.
- Stytch tenant policy state: no Stytch resources are created or altered; nothing to roll back.
