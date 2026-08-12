# Design: add-palette-ai-actions

## Context

The ⌘K palette (`components/common/command-palette.tsx`) renders navigation destinations from `lib/command-registry.ts` (`CommandDestination{id,title,url,section,icon,keywords?}`) plus a search mode over `searchContacts`; selection calls `onNavigate(url)` (router.push) and closes. `useGlobalShortcuts` handles `g <key>` over the same registry. The palette store (`lib/stores/command-palette-store.ts`) is zustand with `{open, mode, session}`.

The AI surfaces that exist today: knowledge assistant (`/dashboard/knowledge`, `knowledge-content.tsx` with local `handleNewChat` at :150-153 — `setCurrentSessionId(null); setOptimisticMessages([])`), AI campaign audience builder (`/dashboard/crm?view=campanas` — the campaigns tab uses the `?view=` routing convention, e.g. feature-gating specs), and inbox context summaries.

Verified facts (premise validation, 2026-08-11):
- `command-registry.ts` — navigation-only destinations; global shortcuts iterate the same list.
- `command-palette.tsx` — `PaletteBody` filters destinations by query and groups by `section`; `onNavigate` closes the palette.
- `knowledge-content.tsx:150-153` — `handleNewChat` is component-local; no store/event bridge exists.
- `command-palette.test.tsx` exists (palette open/close, filtering, search, Enter navigation).
- Campaigns tab route convention: `/dashboard/crm?view=campanas`.

## Goals / Non-Goals

**Goals:**
- IA group in the palette with three working actions (assistant nav, real new-chat reset, audience builder nav).
- Zero impact on navigation destinations, `g <key>` shortcuts, or search mode.
- Deterministic, testable behavior.

**Non-Goals:**
- No new AI backend capability; no backend changes at all.
- No rework of the palette's filtering/navigation internals beyond adding a group.

## Decisions

### D1: Parallel `AiAction` registry, not an extension of `CommandDestination`

`lib/command-registry.ts` gains `AiAction {id, title, section: "IA", icon, keywords?, onSelect: () => void}` + `aiActionRegistry`. Rationale: `CommandDestination.url` is consumed by `useGlobalShortcuts` (g-key navigation); adding actions to it would risk shortcut/filter regressions. A parallel typed registry keeps the palette's execution explicit and the shortcuts untouched. Alternative (union type in CommandDestination) rejected as higher regression risk for zero gain.

### D2: Actions are `onSelect` callbacks (nav + optional side effect)

`aiActionRegistry` entries receive `useRouter`-free callback shape — the palette provides `router` to the group renderer:
1. `Preguntar al asistente` → `router.push("/dashboard/knowledge")`
2. `Nueva conversación de IA` → `requestNewAiChat()` (store) + `router.push("/dashboard/knowledge")`
3. `Audiencia IA para campañas` → `router.push("/dashboard/crm?view=campanas")`

Selection closes the palette (same as navigation).

### D3: New-chat bridge via the palette store signal

`command-palette-store.ts` gains `aiNewChatSignal: number` + `requestNewAiChat()` (increments). `knowledge-content.tsx` subscribes with a `useEffect` on `aiNewChatSignal` (skip initial 0) → calls its existing `handleNewChat`. Rationale: `handleNewChat` is component-local and the knowledge page may not be mounted when the palette acts — the signal is replayed on mount (monotonic counter, not a boolean), so navigating to the page then resets. Alternatives: URL param (`useSearchParams` requires a Suspense boundary in Next 16 client components — added build risk); a bare `CustomEvent` (untyped, no replay semantics) — both rejected.

### D4: IA group renders above navigation, same filtering

`PaletteBody` computes an `aiFiltered` list from `aiActionRegistry` with the same query match, renders `<CommandGroup heading={ui.palette.iaGroup}>` before the navigation groups, and executes `onSelect` then closes. Existing navigation/search behavior untouched.

### D5: Copy under `ui` (Spanish-first)

`ui.palette` keys: `iaGroup`, `askAssistant`, `newAiChat`, `aiCampaignAudience` + keywords (`asistente`, `chat`, `ia`, `campañas`, `audiencia`) (+ `en` mirror).

## Risks / Trade-offs

- **Signal replay semantics**: monotonic counter means a stale "new chat" request fires once on next knowledge mount — intended (the user asked for it). No risk of loops (effect only calls handleNewChat, which doesn't re-trigger the signal).
- **Route drift**: `/dashboard/crm?view=campanas` relies on the tabs convention; if the campaigns tab id changes, only the registry entry needs updating (single source).
- **Palette tests**: existing tests must stay green; new group is additive (empty query shows the IA group — may shift snapshot-style assertions; tests updated accordingly).
