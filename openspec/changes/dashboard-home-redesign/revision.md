# Revision 1 (2026-08-12)

Covers the council verdict `STATUS: REJECTED` / `MARKET: FAIL` (5 numbered required design changes).

- [x] item 1 — `design.md`: added `## Market & Unit Economics` (zero marginal AI cost / LLM actions; zero pricing-plan-credit impact; payment verification preserved byte-for-byte; activation metric scoped as follow-up; honest-data rule as declared differentiator)
- [x] item 2 — `design.md`: added `## Market Risk` with residuals R1 (channel/agent-claims drift, owner product/ops), R2 (activation regression from checklist folding, owner product), R3 (perceived-value gap / competitive substitution, owner product/GTM), R4 (Siigo invoice-data expectation, owner product/backend), each with named risk, owner, trigger, and mitigation
- [x] item 3 — `specs/dashboard-template-restyle/spec.md`: added scenario "Widgets heredan el gate de su superficie fuente" (per-widget permission/entitlement inheritance, no privilege widening on home); `design.md` Decision 8 adds the widget→gate table; `tasks.md` 1.11 adds the gate verification task
- [x] item 4 — `specs/dashboard-template-restyle/spec.md` (requirement "Minimización de datos personales en el panel de conversaciones") + `specs/inbox-ui/spec.md` (scenario "El panel solo expone datos a nivel de snippet"): snippet-only content, no full message bodies/thread/export on home; `design.md` Decision 9; `tasks.md` 1.2/2.3 verification
- [x] item 5 — `specs/dashboard-template-restyle/spec.md`: recomposition SHALL live in `DashboardHome` (`app/dashboard/components/dashboard-home.tsx`); `app/dashboard/page.tsx` SHALL keep only the payment-verification branches; scenario "Verificación de parámetros de pago preservada" updated accordingly

## Artifacts revised in this pass

- `design.md` — market sections added; Decisions 5/8/9 refined (onboarding collapsible per verified auto-hide behavior, RBAC gate table, PII minimization); verified facts added to Context (page.tsx branches, FirstRunChecklist `return null`, no campaigns route today); Open Question on checklist completion state resolved (answer: yes, derived from existing queries); Broadcast route question kept as tasks 1.6 checkpoint.
- `specs/dashboard-template-restyle/spec.md` — component attribution fixed (item 5); added RBAC-gate scenario (item 3) and PII-minimization requirement (item 4).
- `specs/inbox-ui/spec.md` — added snippet-only data-minimization scenario (item 4).
- `proposal.md` — Assumptions updated with verified facts (no campaigns route today → D2.2 omit fallback; checklist auto-hides when complete; zero economic delta); onboarding bullet aligned with the collapsible decision.
- `tasks.md` — 1.2 snippet-only; 1.6 Broadcast route checkpoint; 1.9 collapsible per verified behavior; 1.10 no autonomous-mode claims in banner copy; 1.11 per-widget RBAC gate verification; 2.3 snippet/PII check in Playwright gate.
- `routing.json` — created (advisory): `requires_council: true` (market-in-scope: onboarding/activation + agent display + WhatsApp channel display), `requires_playwright: true` (UI recomposition), `requires_iso: true`, `complexity: medium`.

No verdict items left as residual; the residual risks recorded are the four named market risks (R1–R4) now carried in `design.md` `## Market Risk` for council re-review.
