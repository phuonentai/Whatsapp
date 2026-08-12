# Design: app-shell-modernization

## Context

Frontend-only change in `next_b2b_starter/`. Current shell gaps (verified during gap analysis):

- No command palette, no global search; `searchContacts` server route exists (`go-b2b-starter` crm-repository.ts:54) but is unused by the UI.
- Dark mode tokens exist (`globals.css:35-60`, `darkMode: ['class']` in tailwind.config.ts:4) but only one `dark:` variant exists and nothing toggles the class; shell/auth use hardcoded `gray-*`/`bg-white`.
- `/dashboard` redirects to settings (`app/dashboard/page.tsx:50`); recharts installed but unused.
- No `loading.tsx`, no Suspense; no skip link, no `aria-current`.
- Brand drift: "Your App" (layout.tsx:32, sidebar.tsx:171), "AP Cash" (settings/layout.tsx:4), teal `#0FA8A0` (not-found.tsx:22).
- framer-motion installed, unused — usable for palette/painless motion.

Spec contract: specs/app-shell/spec.md (new) + deltas for admin-panel-navigation, settings-ui.

## Goals / Non-Goals

**Goals**: ⌘K palette, header global search wired to `searchContacts`, dark mode, real dashboard home, global shortcuts, route-level loading, skip link + aria-current, unified brand tokens.

**Non-Goals**: no backend search work beyond consuming the existing route; no i18n; no landing page; no notification center (banner-based alerts stay).

## Decisions

### D1: `cmdk` for the palette (new dependency)
`cmdk` (Comand) is the de-facto standard command palette for React/Next (used by shadcn/ui command component, Vercel, Linear-like apps). Wrapped in a shadcn-style `Command` component under `components/ui/command.tsx`.
- Alternatives: in-house palette — rejected: focus/keyboard/scroll handling is subtle and already solved; build-your-own invites a11y bugs.
- Alternatives: `kbar` — heavier, opinionated router coupling. cmdk is smaller and render-prop friendly.

### D2: Dark mode via `next-themes` + token migration
`next-themes` `ThemeProvider` with `attribute="class"`, `defaultTheme="system"`. `.dark` variable set in globals.css already exists — populate missing tokens. Migrate hardcoded shell/auth colors to `bg-background`/`border`/`text-foreground` tokens (sidebar.tsx, header.tsx, auth pages, dashboard-layout).
- Alternatives: hand-rolled provider — rejected: next-themes handles FOUC + system sync.
- Scope guard: migrate the shell + auth + shared components only; deep component-level `gray-*` in CRM/inbox stay (separate cleanup), but tokens render first.

### D3: Search mode inside the palette
One component, two modes: command mode (nav destinations + actions) and search mode (results from `GET /api/crm/search/contactos` via the existing `searchContacts` query). `Cmd+K` opens command mode; typing a query promotes to search mode after a 300ms debounce.
- Alternatives: separate header search input — rejected: one surface keeps implementation and shortcut model single.

### D4: Dashboard home built from existing queries
New `app/dashboard/home` composition at `/dashboard` (keep route; the current page.tsx handles payment-param verify then renders home instead of redirecting). KPI cards: open conversations (inbox query), contacts count, deals by stage (CRM queries), usage/progress from subscription query. Quick actions: Inbox, CRM, Knowledge, Settings. Skeletons via the change's loading.tsx.
- Alternatives: separate `/home` route — rejected: existing links/redirects target `/dashboard`.

### D5: Global shortcuts via a single `useGlobalShortcuts` hook
Single `keydown` listener on the shell layout, guarded by `isTypingTarget` check (input/textarea/contenteditable). Handles `g d|i|c|k|s`, `?`, `Cmd/Ctrl+K`. Shortcut help overlay renders from the same registry object the hook consumes.
- Alternatives: per-component listeners — rejected: registry keeps shortcuts discoverable and the `?` overlay free.

### D6: Route-level loading via `loading.tsx` files
`app/loading.tsx` (shell skeleton) + `app/dashboard/loading.tsx` + `app/dashboard/(feature)/loading.tsx` skeletons matching each page's shape; rely on Next 16 streaming. No manual Suspense needed for routes (loading.tsx is the Suspense boundary).
- Alternatives: Suspense manually per page — rejected: redundant with loading.tsx.

### D7: Brand tokens consolidated in one constants file
`lib/brand.ts` exporting `PRODUCT_NAME`, `BRAND_PRIMARY` (token ref, not hex); replace "Your App"/"AP Cash"/teal hardcodes (layout.tsx, sidebar.tsx, settings/layout.tsx metadata, not-found.tsx). Keep not-found structure, swap tokens.

## Risks / Trade-offs

- [cmdk + next-themes new deps] → both stable, minimal; pinned versions; tree-shaken.
- [Dark mode token migration touches many files, drift risk] → scope to shell/auth/shared; verify with screenshot pass; remaining `gray-*` tracked as follow-up.
- [searchContacts route may expect specific query shape] → read handler before wiring; debounce + error state in palette if it 5xxs.
- [Dashboard home adds query fan-out] → reuse existing hooks/data already fetched by layout where possible; skeletons prevent layout shift.
- [Keyboard shortcuts conflict with app shortcuts (Enter-to-send, kanban)] → typing-target guard covers inputs; kanban drag unaffected.

## Migration Plan

1. Add deps (`cmdk`, `next-themes`) + `Command` component + ThemeProvider.
2. Token migration + brand constants (visible change, verify per page).
3. Palette + search + shortcuts (built on registry).
4. Dashboard home + loading.tsx files.
5. a11y: skip link, aria-current (sidebar), drawer Escape/focus.
6. Rollback: git revert per commit; localStorage theme pref cleared on revert is harmless; no API/schema changes.

## Open Questions

- ~~Final brand name for PRODUCT_NAME~~ **Resolved during apply:** placeholder `NexoChat` used in `lib/brand.ts` (chosen name; `.com`/`.co` availability unconfirmed at apply time). Update `lib/brand.ts` when the domain/name is finalized.
- ~~Search route exact response contract~~ **Resolved during apply:** `searchContacts` (`GET /crm/contactos/search?q=...`) returns `ContactDto[]` via `crm-repository.ts:54`; unwrapped by the repository. Palette search mode queries it directly.

