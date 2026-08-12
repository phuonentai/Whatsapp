# Proposal: add-context-to-note

## Why

The AI context panel (`conversation-context-panel.tsx`) shows a conversation summary, detected intent, and key facts — but it's a dead-end readout: the natural CRM action, appending that context to the contact's timeline, requires the user to copy-paste into the contact's note form manually. One click should enrich the CRM record.

## What Changes

- **Frontend only**: `ConversationContextPanel` gains an optional `contactId` prop; when full context is available **and** `contactId` is present, it renders a "Guardar como nota" action. The inbox page (`app/dashboard/inbox/page.tsx:171`) passes `selectedConv.contact_id` (already in scope; `ConversationDto.contact_id` confirmed).
- Clicking the action calls the **existing** `createActivity` mutation (`crmRepository.createActivity` → `POST /api/crm/actividades`) with `tipo: "nota"`, a fixed Spanish subject, and `contenido` = formatted summary + intent + key facts. No new backend endpoint.
- Success → confirmation toast and the action is disabled (prevents duplicate notes); failure → error toast, nothing persisted.
- The action is **never rendered** in loading, unavailable, consent-gated, or structural-only states (the summary itself is only produced under consent — PII-free by the context contract).
- Copy under `ui.agent` (Spanish-first + `en` mirror).

## Capabilities

### New Capabilities

- `conversation-context-note`: one-click save of the AI conversation context (summary/intent/key facts) as a CRM contact note — shown only for full consent-gated context, reusing the existing activity endpoint, never persisting on failure.

### Modified Capabilities

None.

## Impact

- **Code**: `next_b2b_starter/` — `components/agent/conversation-context-panel.tsx` (prop + action), `app/dashboard/inbox/page.tsx` (pass `contactId`), `lib/copy/ui.ts` (`ui.agent` keys), component tests. **Zero backend changes** (`POST /crm/actividades` + permission model unchanged).
- **Dependencies**: none new.
- **Systems**: CRM activities timeline (contact-scoped, tipo `nota`) — same record type users already create manually from the contact detail page.

## Non-Goals

- No new backend endpoint, no DB schema, no permission changes (reuses `org:manage`-gated activity creation as today).
- No auto-save (context changes do NOT auto-create notes — only the explicit click).
- No changes to the context generation, consent state machine, or PII masking.
- No local credential storage; no credentials involved.

## Rollback

- **Git state**: revert the touched files (`conversation-context-panel.tsx`, `inbox/page.tsx`, `lib/copy/ui.ts`, component test, this change's artifacts). All additions are additive; no migration, no data.
- **Stytch tenant policy state**: no policy changes, so no policy rollback required.
