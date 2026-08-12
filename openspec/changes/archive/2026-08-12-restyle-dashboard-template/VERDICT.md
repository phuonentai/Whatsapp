# Verdict — restyle-dashboard-template

STATUS: APPROVED

Independent adversarial re-review (security / DBA / SRE) of `design.md`, the delta specs, and `tasks.md` after the design rework. All factual premises re-verified against the repo at review time (2026-08-12). The four required changes from the first verdict are confirmed implemented in `design.md` D2.1/D2.2/D3, the delta specs, and the code. One new minor spec-coherence finding (A) is recorded below; it does not block approval but must be resolved before archive.

## Staff Security Engineer

- **Payment-verification branches — RESOLVED.** Verified in repo: `app/dashboard/page.tsx` (server component) runs `verifyPayment(checkoutId)` for Polar `checkout_id`, the preapproval-only settlement redirect for `preapproval_id` without `payment_id`, and `verifyMercadoPagoPayment({ paymentId })` for `payment_id`/`preference_id` — then renders `<DashboardHome />`. The change did NOT touch the page (`page.tsx` unchanged; verified). `app/dashboard/page.test.tsx` covers all four routing cases (4/4): preapproval-only, preapproval+payment, payment-only error, home render. Business-critical regression risk closed.
- **RBAC-aware template navigation — RESOLVED.** `components/layout/sidebar.tsx` filters every item by `permissions.permissions.includes(item.permission)` and/or `entitlementKeys` from `useEntitlementQuery`. The "Inteligencia Artificial" group renders only real routes (Copiloto IA → `/dashboard/settings?view=ai`, gated `org:manage`); Entrenamiento/Automatizaciones are deliberately not rendered (no product route — documented decision D2.2); the "IA Insights" card and its CTA are gated by `org:manage`. No unconditional links to gated or nonexistent sections.
- **No new attack surface.** UI-only change: no new dependencies, no secrets, no Stytch contract change. `hooks/use-signup-flow.ts` is untouched (same `signupRepository.createOrganizationWithMagicLink` call, same validations, 3-step flow); `BusinessContextStep` keeps `saveBusinessContext + onContinue` intact. The notifications bell routes to a real surface (`/dashboard/settings?view=audit`) instead of invented UI; avatar uses real profile initials (`user-menu.tsx`), no fabricated names.
- **LOW (non-blocking):** top-bar ⌘K search relies on the pre-existing command palette (app-shell requirement, unchanged). If a dedicated notifications surface ships later, it must be RBAC-gated like the rest of the shell.

## Staff DBA

- **N/A on schema.** No migrations, no SQLC, no backend changes (proposal Impact + Phase 0 baseline confirm). Nothing to review at the storage layer.
- **Fan-out — RESOLVED.** `app/dashboard/components/dashboard-home.tsx` KPI queries are exactly the ones `DashboardHome` already consumed: `useConversationsQuery`, CRM queries, and `analyticsRepository.revenue({ period })` — the reports module's existing query, gated by `analytics_module` entitlement + `invoice:view` permission, with `staleTime` and skeleton loading. No new queries, no fan-out.
- **"—" rule — RESOLVED and enforced in code.** Ventas semana renders "—" without the analytics module/permission; Facturas emitidas and Tiempo respuesta IA render "—" (documented as having no data source, never fabricated); `components/inbox-metrics.tsx` (at `app/dashboard/inbox/components/`) renders real counts for conversaciones hoy / por responder and "—" for tasa de respuesta / tiempo promedio. Delta spec scenarios encode the rule.
- **LOW (non-blocking):** the revenue aggregation adds one existing-query call per dashboard render for entitled orgs only; the loading skeleton covers latency. No action required.

## SRE

- **Rollback — RESOLVED.** Git revert only; UI-only, no Stytch/tenant state, no migrations. Consistent with the proposal.
- **Verification gate — honest and adequately recorded.** `tasks.md` §5.1 records: `pnpm lint` PASS (0 errors / 4 pre-existing warnings), `pnpm build` PASS (Next 16.0.10 production), `npx tsc --noEmit` PASS, unit tests 10/10 (`dashboard/page` 4/4, `signup/page` 3/3, `conversation-list` 3/3), `openspec validate` PASS, and a working smoke on port 3100 (2026-08-12: `/signup` HTTP 200 with SSR slate-900 wizard markup; `/dashboard` + `/dashboard/inbox` 307 to `/auth?returnTo=...` — expected without a Stytch session). The shared 3001 dev server's Turbopack worker-spawn failure is documented as affecting untouched routes too, with a Phase 0 repo-wide baseline anchoring the tree state.
- **Checkout-return verification — now explicit in the gate** (Polar `checkout_id` / MercadoPago `payment_id`/`preapproval_id` → `/dashboard/settings?view=subscription` with `payment_verified=true`/`payment_error=true`), unit-covered; full-stack e2e PENDIENTE.
- **Pending-infra items are marked, not dropped.** Smoke dev on the healthy port, Playwright e2e (`inbox-ui`, `auth-passwordless`, `admin-panel`) and full-stack checkout returns remain PENDIENTE. Per the AGENTS.md verification gate these MUST pass before archive; `tasks.md` §5.2 correctly defers archive for that reason (the outdated "REJECTED" rationale from the previous review has been replaced).
- **LOW (non-blocking):** re-confirm the Turbopack dev-server failure reproduces outside this workspace before relying on `pnpm build` alone as the render proxy.

## New finding (spec-coherence, resolve before archive)

- **A. Delta spec "Bandeja" filter wording vs implemented model.** `openspec/changes/restyle-dashboard-template/specs/dashboard-template-restyle/spec.md` (and the synced living spec) requires a toolbar with "filtros por estado/agente" and a scenario "los filtros de estado/agente SHALL filtrar la lista según la lógica existente". The implementation (`app/dashboard/inbox/page.tsx` + `conversation-list.tsx`) filters by channel and status tabs; there is NO agent filter because the `Conversation` model does not expose an agent — documented in tasks §3.1 ("sin agente porque el modelo de conversación no expone agente"). Per AGENTS.md the spec is authoritative; here the spec text is genuinely wrong relative to the data model. **Required:** amend the delta spec requirement/scenario to reflect the implemented filters (canal + estado; agente solo si el modelo lo expone) and re-sync to the living spec. Non-blocking for design approval — presentation-layer, no behavioral risk.
- **B. LOW (follow-up, not required):** `components/layout/header.tsx` derives breadcrumbs/page titles from pathname segments (e.g., `/dashboard/inbox` → "Inbox") while the sidebar says "Conversaciones" and the page title is "Bandeja de entrada" (`ui.inbox.title`). Pre-existing breadcrumb logic, not introduced here; a copy-consistency follow-up is recommended (settings views such as `?view=siigo` also render raw labels).

## Conditions for archive (must be satisfied before `/opsx-archive`)

1. Resolve finding A: amend the delta spec filter wording to match the implemented channel/status model and re-sync the living spec.
2. All verification tasks currently PENDIENTE in §5.1 pass: smoke `pnpm dev` on `/dashboard`, `/dashboard/inbox`, `/signup` (healthy server); Playwright e2e `inbox-ui`, `auth-passwordless`, `admin-panel`; full-stack checkout-return e2e (Polar + MercadoPago). The recorded PASSes (lint/build/tsc/unit 10/10/openspec validate) unblock design approval only — not archive.
3. Page-object updates in §3.2 must be exercised by the pending e2e run (presentation-layer-only; no logic changed — acceptable).

## Required design changes (from the prior verdict) — status

1. Preserve payment-verification branches + state `DashboardHome` content reuse — **DONE** (D2.1, delta spec, task 2.1, `page.tsx` verified untouched, tests 4/4).
2. Add checkout-return verification to the gate — **DONE** (task 5.1, explicit; unit-covered, e2e pending infra).
3. RBAC-gated template nav groups / IA Insights card — **DONE** (D2.2, delta spec scenario, `sidebar.tsx` verified).
4. Reuse existing queries, no fan-out, "—" rule — **DONE** (D3, delta spec scenarios, code verified).
