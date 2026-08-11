# Design: fix-e2e-integration-tests

## Context

The backend message contract was migrated from `whatsapp_message_id` to `provider_message_id` (committed: `go-b2b-starter/internal/db/postgres/sqlc/query/crm.sql`, generated sqlc models, `crm/domain/message.go` JSON tag `provider_message_id,omitempty`, and `next_b2b_starter/lib/api/api/dto/conversation.dto.ts:23`). The WhatsApp e2e specs still assert against the old field name:

- `next_b2b_starter/e2e/specs/whatsapp-edge-cases.spec.ts:25` (local `MessageDto` interface) and `:44` (`findMessageByWhatsappId` filter)
- `next_b2b_starter/e2e/specs/whatsapp-inbox.spec.ts:25` and `:105` (duplicate-delivery filter)

The three affected tests fail deterministically because the API response never contains `whatsapp_message_id`. `inbox-ui.spec.ts` also ships mocked payloads with the stale field (lines 75/97/110). The living `crm-whatsapp-e2e` spec repeats the stale name in the duplicate-delivery requirement.

## Goals / Non-Goals

**Goals:**
- Make the three WhatsApp message-persistence e2e tests pass by asserting on the field the API actually returns (`provider_message_id`).
- Keep e2e payload mocks consistent with the real `MessageDto` shape.
- Reconcile the `crm-whatsapp-e2e` living spec with the implemented contract.

**Non-Goals:**
- No backend, frontend application, API, or schema changes.
- No changes to `agent`/`campaign` DTOs that still use `whatsapp_message_id` (separate contract).
- No ownership of the Siigo e2e spec or flaky deals/tags tests (environment/DB-state dependent; owned by their changes).

## Decisions

### D1: e2e assertions filter by `provider_message_id`
Rename the field in the two failing specs' local `MessageDto` interfaces and filter expressions to `provider_message_id`, matching the committed backend API and FE `MessageDto`. No backend change is needed — the source of truth (sqlc + domain JSON) already emits `provider_message_id`.
- Alternatives: add a backward-compat alias `whatsapp_message_id` in the API — rejected: the rename is intentional and committed; re-adding a deprecated alias would entrench the stale contract.

### D2: Align `inbox-ui.spec.ts` mocked payloads to the real DTO
The UI-level mock responses use `whatsapp_message_id`; the FE `MessageDto` reads `provider_message_id`. Rendering doesn't depend on the id field, so this is consistency-only, but aligning avoids future confusion when the id field is displayed or asserted.
- Alternatives: leave mocks as-is — rejected: keeps a known-stale field name in the suite.

### D3: Update `crm-whatsapp-e2e` spec wording
The duplicate-delivery requirement SHALL name `provider_message_id`, the implemented persistence key. This is a spec-alignment delta (behavior unchanged; the wording already described the wrong field).

### D4: Verify under the canonical e2e stack
Targeted verification runs `playwright` against the canonical `make test-e2e` bootstrap (fresh `saas_db_test` reset+migrated+seeded, mock Siigo on `:8090`, `AUTH_MOCK_ENABLED=true` backend on `:8080`, frontend on `:3001`). This isolates the three field-name fixes from the environment-dependent Siigo/deals/tags failures observed in ad-hoc runs.

## Risks / Trade-offs

- [Field rename might hide a deeper persistence bug] → Verified via direct inspection that the API returns `provider_message_id` (sqlc gen 14×/0×, domain JSON tag); the tests then exercise real end-to-end persistence.
- [Siigo/flaky failures could reappear in the full gate run] → Out of scope; recorded as env/DB-state dependent. Final gate records pass/fail and, if only pre-existing backend-integration failures remain, an explicit exception/archive decision is recorded.
- [Inbox-ui mock alignment is cosmetic] → Low risk, single-field rename in three mock objects; UI assertions unchanged.

## Migration Plan

1. Rename field in `whatsapp-edge-cases.spec.ts` and `whatsapp-inbox.spec.ts` (interface + filters).
2. Rename field in `inbox-ui.spec.ts` mock payloads.
3. Write the `crm-whatsapp-e2e` delta spec wording.
4. Verify: eslint + `npx tsc --noEmit` on the touched specs; targeted playwright run of the two whatsapp spec files under the canonical stack; `grep` for stale `whatsapp_message_id` in `e2e/specs` (must be empty); full `make test-e2e` gate.
5. Rollback: `git revert` of the change's commits; no migrations, no Stytch state.

## Open Questions

- None. Field-name mismatch and fix are fully confirmed against committed code.
