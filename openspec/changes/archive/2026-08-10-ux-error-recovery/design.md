# Design: ux-error-recovery

## Context

Frontend-only change in `next_b2b_starter/`. Current failure UX is inconsistent and often silent:

- Query failures render eternal "Cargando..." (`ticket-detail.tsx:27`, `contact-table.tsx:62`) — no `isError` branches anywhere in CRM/tickets.
- `use-send-message.ts` has no `onError`; `reply-input.tsx:31-36` awaits without try/catch → unhandled rejection, silent failure.
- Native `window.alert`/`confirm` in `plans-modal.tsx:73,100`, `member-list.tsx:227-229`, `compliance-section.tsx:19-65`.
- No unread indicators in inbox conversation list; no `aria-live` for message arrival or streaming.
- Existing good patterns to reuse: `ConfirmDialog` (components/crm/confirm-dialog.tsx), sonner toasts, section-level error alerts in settings (`settings-content.tsx:822-833`), `Skeleton` component.

Constraints: TanStack Query is the data layer; sonner the toast layer; no new runtime dependencies needed. Spec contract: specs/ui-error-recovery/spec.md + deltas for inbox-ui, crm-frontend, billing-provider-ux, settings-ui.

## Goals / Non-Goals

**Goals**: every data view distinguishes loading/error/success with retry; no silent mutation failures; no native browser dialogs in product flows; unread indicators; live-region announcements.

**Non-Goals**: no backend changes, no auth/Stytch changes, no i18n extraction, no optimistic updates (ui-data-tables change), no message real-time transport change (polling stays).

## Decisions

### D1: Shared `ErrorState` component over per-view error UI
One component `components/common/error-state.tsx` with props `title`, `description`, `onRetry`, `isRetrying`. Rendered inline where the view body would be.
- Alternatives: per-view bespoke alerts — rejected: proven drift in settings sections already.
- Rationale: single retry affordance + consistent copy; matches spec "shared error-state component".

### D2: Enforce error branches at the query-hook level with a small pattern, not a new library
Every data view gets the pattern: `const { data, isLoading, isError, refetch, isRefetching } = useQuery(...)` → `if (isLoading) return <Skeleton/>` / `if (isError) return <ErrorState onRetry={refetch}/>`.
- Alternatives: react-query error boundaries, error-boundary components — rejected as global/interstitial; inline states preserve page context and match existing settings pattern.
- Rationale: zero new deps, follows existing conventions, satisfies spec scenarios.

### D3: Send failure handled in the UI layer, not the mutation
`reply-input.tsx` wraps send in try/catch: on throw → `toast.error` (Spanish/English per file language — see open question) and keep draft (do NOT clear before await; only clear on success).
- Alternatives: `onError` on mutation — rejected because the current call site uses `mutateAsync` awaiting in `page.tsx:65-68`; catching at call site guarantees no unhandled rejection regardless of future callers.
- Rationale: spec requires no unhandled promise rejection; catching at the single call site is the minimal correct fix.

### D4: Custom dialogs for all native dialog sites
- `plans-modal.tsx`: replace `window.alert("cancel current subscription first")` with an inline banner + confirm dialog reusing `ConfirmDialog`.
- `member-list.tsx`: role-change and removal confirmations → `ConfirmDialog` (already exists; current code uses `confirm()`).
- `compliance-section.tsx`: forget flow → `ConfirmDialog` (export stays as-is).
- Rationale: one shared dialog primitive already built and tested; native dialogs are inaccessible and block page.

### D5: Unread indicators computed client-side from message data
Track a `lastSeenAt` map per conversation in the inbox page (zustand or component state). A conversation is unread when its latest inbound message timestamp > lastSeenAt. Clearing on open/send writes to state.
- Alternatives: backend `last_read_at` column — rejected: requires DB migration + API work, out of scope; client-side derived state satisfies the spec for a polling UI and is honest about eventual refresh.
- Note: this is ephemeral per-device state; acceptable for v1, flag as open question.

### D6: Live regions added at container level
Message thread container gets `role="log" aria-live="polite"`; knowledge chat assistant container gets `aria-live="polite"` (throttled via SSE token accumulation — one DOM update per rendered frame, existing behavior). Suggestion panel updates announced via an `aria-live="polite"` region.
- Rationale: container-level regions announce changes without per-item wiring; existing streaming already accumulates tokens so no per-token spam.

## Risks / Trade-offs

- [Unread state is per-device and resets on browser change] → acceptable v1; document as open question; backend persistence deferred.
- [Adding error branches touches many components — risk of missed spots] → tasks enumerate every known view; a grep for `isLoading` without `isError` is a verification step.
- [Draft preservation changes send flow ordering] → clear input ONLY after successful await; unit test covers failure path (`reply-input.test.tsx` exists).
- [ConfirmDialog reuse changes copy] → keep existing Spanish strings in the dialog.

## Migration Plan

1. Land shared `ErrorState` + dialog swaps (independent, low risk).
2. Land send-failure + unread + live regions.
3. Sweep all data views for error/retry branches.
4. Rollback: git revert per commit; all changes additive UI, no schema/API changes, no Stytch state.

## Open Questions

- Language consistency for new strings: inbox is English, CRM/settings Spanish. Per-file language for now (i18n is a separate change).
- Should unread state persist per device (localStorage)? Currently in-memory; localStorage is a cheap follow-up.
