## Context

MVP launch gate passed, but three code-level gaps remain: (1) the billing UI is Polar-only — the MercadoPago cancel server action (`lib/actions/billing/cancel-mp-subscription.ts`) and backend route `POST /api/subscriptions/mp-cancel` (`internal/modules/billing/routes.go:28`) exist but no component calls them; (2) `subscription-paywall.tsx` has no provider handling; (3) several tasks in in-flight changes are open although the code is present and verified. Two E2E specs are flaky under parallel load. This change is UI-only and additive; no backend, DB, auth, or CI config changes.

## Goals / Non-Goals

**Goals:**
- Provider-aware cancel + copy in the two billing components, reusing existing actions/routes
- Reconcile stale task checkboxes where code presence is verified
- Harden the two flaky E2E specs
- Record archive-deferral entries for changes with open archive decisions

**Non-Goals:**
- No backend/endpoint/migration changes
- No live-sandbox verification (deployment-gated, already recorded in owning changes)
- No CI live run, no Stytch RBAC changes

## Decisions

**D1: Branch on `isMercadoPagoEnabled()` (env flag), not a backend provider query.**
`lib/mercadopago/config.ts` already centralizes enablement (`NEXT_PUBLIC_MERCADOPAGO_PLAN_ID`). A backend `billing_provider` query exists (SQLC) but is not needed for UI branching; the gate state already surfaces `MP_UNCONFIGURED`/`POLAR_UNCONFIGURED` reasons for configuration errors. Alternative (query provider server-side) rejected: adds a fetch to every settings render for no UX gain; the backend route is the enforcement point.

**D2: Reuse `cancelMPSubscription` action as-is; only add the UI branch.**
Action already enforces Stytch session + `canManageSubscriptions` before the outbound call and posts `{ subscription_id }` to `/api/subscriptions/mp-cancel`. Cancel dialog/copy mirrors the existing Polar dialog to keep UX consistent. Alternative (new action) rejected — duplicate code with no behavioral difference.

**D3: Cancel branch keyed on subscription state, not just env.**
Branch logic: if MercadoPago enabled → MP cancel path; else Polar path. The existing `cancelAtPeriodEnd`/resume affordances remain unchanged; MP cancel returns a status the UI surfaces like the Polar flow.

**D4: E2E hardening targets the two failure modes directly.**
- `whatsapp-inbox.spec.ts:65`: replace the bare `fetch` deliver with `expect.poll`/retry on response status, tolerating a dropped request.
- `deals.spec.ts:91`: `waitForResponse` on `POST /api/crm/negocios` times out because the Next dev server drops the request under parallel load — retry the POST on drop or widen the wait window with a retry loop.
Alternative (retry/repeatEach) rejected: masks real failures instead of fixing the wait pattern.

**D5: Reconciliation edits only where code presence was verified.**
SQLC queries (`organizations.sql.go:372,633`), `test:e2e` (`package.json:9`), GitLab `run-frontend`/`run-e2e` (`go-b2b-starter/.gitlab-ci.yml:22,41`) all verified present in the working tree. Checkboxes flip to done with a verification note; no speculative checkboxes.

## Risks / Trade-offs

- [MP cancel path unverifiable live until deployment] → Unit-level: `tsc`/`pnpm build` green; route + action already covered by backend tests; sandbox e2e already recorded as deferred in `wire-mercadopago-billing` 13.x.
- [E2E spec fixes unverifiable in this env (port 3001 occupied)] → Typecheck-only gate here; full-suite run recorded as deferred, matching existing deferral pattern.
- [Reconciliation checkboxes flipped without a re-run] → Each flip carries a verified-present note; any future failure reopens the task.

## Migration Plan

UI-only, additive: deploy FE, no DB or config steps. Rollback: revert commit. No Stytch tenant policy state affected.

## Open Questions

- None blocking. MP cancel response shape consumed generically (status surfaced like Polar flow); exact copy wording for paywall MP state to follow existing PSE/Nequi phrasing.
