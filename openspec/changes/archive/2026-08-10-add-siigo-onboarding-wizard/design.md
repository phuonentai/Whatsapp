## Context

Changes `add-siigo-org-onboarding` (connections, state machine, kill-switch, assisted provisioning) and `add-siigo-onboarding-data` (numeration, import, sandbox test) deliver the backend API surface. The FE consumes it. Existing FE conventions: settings sections live in `app/dashboard/settings/components/` (`whatsapp-config-section.tsx`, `modules-section.tsx` + `modules-section.test.tsx`), wired through `settings-content.tsx`; TanStack Query + react-hook-form + shadcn/ui in use; Spanish-first copy matching the CRM. No dedicated admin panel page exists in FE (admin-panel specs cover navigation + audit-log view in settings) — the admin onboarding view is the first admin-role surface and follows the settings-section pattern. Known baseline: `pnpm lint` exits 1 at documented 9+1, `tsc --noEmit` blocked by an external uncommitted edit in `lib/auth/stytch/server.ts:178` (recorded in `add-siigo-invoicing` task 10.2).

## Goals / Non-Goals

**Goals:**
- Settings Siigo section with state-driven banner (never silent)
- 5-step wizard, server-persisted progress, resumable
- Branching for has-Siigo / manual / assisted paths
- Kill-switch toggle
- Admin onboarding overview + assisted provisioning form
- Component tests following `modules-section.test.tsx` precedent

**Non-Goals:**
- No invoice list/detail UI (WhatsApp carries the invoice link)
- No product catalog UI
- No new backend endpoints outside changes 1/2
- No multi-language UI

## Decisions

### 1. Section, not separate route

**Chosen:** `siigo-integration-section.tsx` renders inside the existing settings page (tab/section list in `settings-content.tsx`), mirroring `whatsapp-config-section.tsx`. The wizard is an internal state machine over the backend connection state, not a separate page with local-only progress.

**Alternatives considered:** *Dedicated `/dashboard/settings/siigo` route* — rejected: breaks the established one-page-sections pattern; the backend state machine already persists progress, a route adds nothing.

### 2. Backend state is the single source of truth for wizard progress

**Chosen:** Every wizard render derives step completion from `GET /api/v1/org/siigo/status` (connection state); no localStorage; `isPending`/`isFetching` from TanStack Query; actions (connect, confirm numeration, import confirm, test invoice, pause/resume) invalidate the status query on success. Errors from server render verbatim via react-hook-form error mapping.

**Alternatives considered:** *Local wizard state + submit everything at the end* — rejected: dead on refresh and diverges from backend truth mid-onboarding; resumability is a spec requirement.

### 3. Branching as declarative state map

**Chosen:** A single `SiigoStatusView` component switches on connection state: `none` → connect form; `awaiting_setup` → assisted banner; `connected|numeracion_ok|sandbox_ok` → wizard; `paused` → paused banner + resume; `invoicing_disabled` → single-line notice; `live` → active banner + pause toggle + (collapsed) history. One switch, tested exhaustively; no scattered conditionals.

**Alternatives considered:** *Per-state components checking flags* — rejected: drift risk; a state-map with one test per state is the audit surface.

### 4. Admin view as admin-gated section

**Chosen:** Admin onboarding overview + assisted form as an admin-scoped surface (role check via existing auth/RBAC client, mirroring the settings scoping), rendered as a table with per-org rows and the provisioning form inline for `awaiting_setup` rows. Nav entry added following the permission-filtered sidebar pattern from `admin-panel-navigation` specs.

**Alternatives considered:** *Fold into settings per-org only* — rejected: operators must see all orgs and provision creds; the spec (admin-panel-navigation) requires the admin surface.

### 5. Test strategy

**Chosen:** Component tests per state (mock TanStack Query + fetch): banner states, wizard step gating, import preview→confirm sequence, kill-switch toggle, admin deny for non-admin. Match `modules-section.test.tsx` patterns (vitest + RTL as configured in the repo).

**Alternatives considered:** *Full e2e* — rejected for this change: component tests cover the state matrix; e2e deferred to the FE e2e tooling change if present.

## Risks / Trade-offs

- [Backend endpoint shape drifts from FE expectations] → Mitigation: FE consumes only change-1/2 endpoints; any shape gap is fixed in those changes' tasks, and the wizard is written against their documented responses.
- [tsc baseline external breakage (`server.ts:178`)] → Mitigation: recorded external exception (as in `add-siigo-invoicing` 10.2); wizard code itself must pass `tsc --noEmit` scoped checks; full tsc gate deferred to the owning change.
- [Template approval status source not exposed] → Mitigation: display reads WhatsApp config data available today; if unavailable, render "pendiente" with the documented exception rather than inventing a backend call.
- [Admin surface is first of its kind] → Mitigation: reuse settings-section layout + existing RBAC client; keep it small; admin-panel spec deltas scoped to onboarding only.

## Migration Plan

1. `lib/api/` client functions + TanStack Query hooks for: status, connect, confirm-numeration, import preview/confirm, sync, test-invoice, pause/resume, admin provision/list.
2. `SiigoStatusView` state map + banner components + tests (all states).
3. Wizard step components (connect form → numeración → import → sandbox → activate) + tests.
4. Kill-switch toggle + tests.
5. Admin overview + assisted form + nav entry + tests.
6. Wire section into `settings-content.tsx`.
7. Gate: `pnpm lint` (baseline documented), scoped `tsc`, `pnpm build`, component tests; record external exceptions.
8. Rollback: revert commits; unhook section from `settings-content.tsx` (UI-only, backend unaffected).

## Open Questions

- Where exactly does the admin surface live — settings-tab (like audit-log-view) or a new `/dashboard/admin` route? (implementation resolves against repo admin precedent; spec requirement unaffected)
- Should the activation step require a typed confirmation (e.g., checkbox "He verificado la resolución")? (default: simple button; typed confirmation added if compliance asks)
