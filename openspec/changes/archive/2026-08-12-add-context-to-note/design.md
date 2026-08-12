# Design: add-context-to-note

## Context

The AI context panel (`next_b2b_starter/components/agent/conversation-context-panel.tsx`) renders summary/intent/key facts for a conversation and has four states: loading, unavailable, consent-gated/structural, and full. It receives only `conversationId`. The inbox page mounts it at `app/dashboard/inbox/page.tsx:171` with `selectedConv` in scope, and `ConversationDto` carries `contact_id` (`lib/api/api/dto/conversation.dto.ts:4`).

The CRM activity path already exists end to end: `crmRepository.createActivity` → `POST /api/crm/actividades` (`crm-repository.ts:81`), `useCreateActivityMutation` (`use-crm-mutations.ts:117`), and the contact-detail page creates `tipo: "nota"` activities with `{contact_id, tipo, asunto, contenido}` (`contact-detail.tsx:41-44`).

Verified facts (premise validation, 2026-08-11):
- `ConversationDto.contact_id` exists; `selectedConv` is the inbox page's selected conversation.
- `createActivity` payload shape: `{contact_id, tipo: "nota", asunto, contenido}`.
- The context contract is PII-free (agent context generation masks PII in summary/intent/key facts — the panel only renders full context under consent).

## Goals / Non-Goals

**Goals:**
- One click from the context panel creates a contact note with the AI summary.
- Zero backend changes; reuse the existing activity endpoint, mutation, and permission model.
- Safe by default: only full-context state, only when contact id is known, explicit click, no auto-repeat.

**Non-Goals:**
- No new endpoint, schema, or permissions.
- No auto-save on context refresh.
- No changes to context generation, consent gating, or PII masking.

## Decisions

### D1: Frontend-only via an optional `contactId` prop

`ConversationContextPanel` gains `contactId?: number`; the inbox page passes `selectedConv.contact_id`. The action renders only when `data` is full context AND `contactId !== undefined`. Rationale: zero backend surface; the data the action needs already exists client-side. Alternative (new `POST /agent/conversations/:id/note` endpoint) rejected — it would duplicate the activity API and its permission model for no benefit.

### D2: Reuse `useCreateActivityMutation` with the `tipo: "nota"` pattern

Click → `createActivity.mutate({contact_id: contactId, tipo: "nota", asunto: ui.agent.noteSubject, contenido})` where `contenido` is a plain-text composition of the summary + intent + key facts. The mutation's existing invalidate (`queryKeys.crm.all`) refreshes the CRM timeline. Permission model is unchanged (activity creation is `org:manage`-gated as today).

### D3: Post-save state machine

`isPending` disables the button; on success a `saved` local state disables it for the panel's lifetime (duplicate-note guard) + confirmation toast; on error a toast and no state change (retry allowed). Per-conversation, per-mount — navigating away resets it (acceptable: the note exists on the timeline, visible next time).

### D4: Note content and PII posture

Subject: fixed Spanish ("Resumen IA de la conversación"). Content: `Resumen: …\nIntención: …\nDatos clave: …` (plain text, missing sections omitted). The context contract already excludes PII from summary/intent/key facts, so the note inherits that property — documented, no extra masking step.

### D5: Copy under `ui.agent`

Keys: `saveNote`, `noteSubject`, `noteSaved`, `noteError` (+ `en` mirror), Spanish-first.

## Risks / Trade-offs

- **Duplicate notes**: mitigated by the disabled-after-save guard; a hard refresh re-enables it (user can re-save deliberately — acceptable, same as manual notes).
- **Stale context**: the note captures what the panel showed; if context refreshes, the saved note reflects the prior snapshot (documented, matches "explicit click" semantics).
- **Contact-linkage assumption**: inbox conversations always have `contact_id`; if a future conversation type lacks one, the optional prop simply hides the action (no failure mode).
