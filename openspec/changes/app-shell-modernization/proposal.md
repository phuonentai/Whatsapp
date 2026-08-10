# Change Proposal: app-shell-modernization

## Why

The app shell lacks the baseline affordances of a 2026 AI SaaS: no command palette, no global search (server `searchContacts` exists, unused — crm-repository.ts:54), no dark mode (`.dark` tokens defined but never toggled — globals.css:35-60), no dashboard home (redirects straight to settings — dashboard/page.tsx:50), no keyboard shortcuts, and zero route-level `loading.tsx`/Suspense. This is the P1 cluster of the UI/UX gap analysis.

## What Changes

- Add a command palette (Ctrl/Cmd+K) reachable from header and keyboard, navigating to all sidebar destinations and settings views; fuzzy-filtered, keyboard-navigable, a11y-compliant.
- Add global search in the header: opens palette in search mode; for contacts, delegate to the existing `searchContacts` server route; results keyboard-navigable.
- Add dark mode: `next-themes` provider, theme toggle in user menu, persist preference, `prefers-color-scheme` default; migrate shell hardcoded `gray-*`/`bg-white` to token variables so dark theme is coherent.
- Build a real dashboard home at `/dashboard` with KPI cards, recent activity, and quick actions (recharts is already a dependency); keep payment-param verification behavior.
- Add global keyboard shortcuts (g d / g i / g c / g k for nav, ? for shortcuts help).
- Add `loading.tsx` route-level skeletons and Suspense boundaries for app shell and list pages.
- Add skip-to-content link and `aria-current` on active nav items; focus management for mobile drawer (Escape close).
- Unify brand tokens: replace "Your App" / "AP Cash" / teal #0FA8A0 drift with a single brand token set.

## Capabilities

### New Capabilities
- `app-shell`: Command palette, global search, keyboard shortcuts, dark mode, dashboard home, route-level loading, and shell accessibility (skip link, aria-current, drawer focus).

### Modified Capabilities
- `admin-panel-navigation`: sidebar SHALL include Dashboard home entry and aria-current active states; keyboard shortcut navigation to all nav destinations.
- `settings-ui`: settings views SHALL be reachable from the command palette.

## Impact

- Frontend only (`next_b2b_starter/`): `components/layout/*`, `app/dashboard/page.tsx` (new home), new `components/common/command-palette.tsx`, `app/loading.tsx` + per-route `loading.tsx`, `app/layout.tsx` (providers), `globals.css`/`tailwind.config.ts`.
- New dependencies: `next-themes`, `cmdk` (or in-house palette — decide in design).
- No API, schema, or Stytch changes.

## Non-Goals

- No local credential, password, MFA, or session-token storage — Stytch B2B remains the sole identity/session authority; theme preference stored in `localStorage` only.
- No backend search API work beyond wiring the existing `searchContacts` route.
- No landing page rebuild or i18n (separate changes).

## Rollback

- Git state: revert the change's commits; additive UI, no migrations.
- Stytch tenant policy state: no Stytch resources are created or altered; nothing to roll back.
