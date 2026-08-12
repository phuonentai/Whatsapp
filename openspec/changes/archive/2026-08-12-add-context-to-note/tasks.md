# Tasks: add-context-to-note

## 1. Frontend: save-context action [FE-NEXT]

- [x] 1.1 Add optional `contactId?: number` prop to `components/agent/conversation-context-panel.tsx`; render a "Guardar como nota" action only in the full-context branch when `contactId` is defined (never in loading/unavailable/consent-gated/structural states). Verify: `pnpm lint`; `npx tsc --noEmit`.
- [x] 1.2 Wire the click to `useCreateActivityMutation` (`use-crm-mutations.ts`) with `{contact_id, tipo: "nota", asunto: ui.agent.noteSubject, contenido}` where `contenido` composes summary + intent + key facts as plain text (missing sections omitted); pending disables the button, success disables it + confirmation toast, error → toast + retry allowed. Verify: `pnpm lint`; `npx tsc --noEmit`.
- [x] 1.3 Pass `contactId={selectedConv.contact_id}` at the mount site `app/dashboard/inbox/page.tsx:171`. Verify: `npx tsc --noEmit`.
- [x] 1.4 Add `ui.agent` copy keys in `lib/copy/ui.ts` (+ `en` mirror): `saveNote`, `noteSubject`, `noteSaved`, `noteError`. Verify: `pnpm lint`; keys referenced from the component.
- [x] 1.5 Component tests (`conversation-context-panel.test.tsx`): button renders only on full context with contactId; structural/consent-gated/loading states hide it; click creates the activity with correct contact_id + content; failure shows toast and no mutation success path; post-save disabled. Verify: `pnpm exec vitest run components/agent/conversation-context-panel.test.tsx` — passes.

## 2. Verification gate [OPS-GOV]

- [x] 2.1 Run frontend gate: `pnpm lint` (0 errors, pre-existing warnings acceptable) and `npx tsc --noEmit`. Verify: both pass; record results here.
- [x] 2.2 Run affected component tests: `pnpm exec vitest run components/agent/conversation-context-panel.test.tsx`. Verify: passes; record results.
- [x] 2.3 Record results and archive decision (`/opsx-archive` or `**Archive deferred:** <reason>`) in this file. Verify: entry present.

## Verification record (2026-08-11)

| Gate | Command | Result |
| --- | --- | --- |
| Lint | `pnpm lint` | Pass — 0 errors, 4 pre-existing warnings (company-table.tsx, contact-table.tsx, deal-kanban.tsx, marketing/prose.tsx) |
| Typecheck | `npx tsc --noEmit` | Pass (exit 0) |
| Component tests | `pnpm exec vitest run components/agent/conversation-context-panel.test.tsx` | Pass — 13/13 tests |
| Build (bonus) | `pnpm build` | Pass (Next.js 16.0.10, exit 0) — no `.next` lock contention |

**Archive deferred:** centralized verification phase per repo practice
