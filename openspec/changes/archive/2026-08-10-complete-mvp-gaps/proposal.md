## Why

The mvp-launch change passed its verification gate (backend build/vet/test, frontend build/tsc, lint at documented baseline), but three code-level MVP gaps remained unfinished and are now the only open code items: (a) the billing UI is not provider-aware — the settings subscription tab cancels via Polar only, while the MercadoPago cancel server action (`lib/actions/billing/cancel-mp-subscription.ts`) and its backend route (`POST /api/subscriptions/mp-cancel`) exist but are wired to no component; (b) the subscription paywall shows Polar-only copy in its inactive state, with no provider handling; (c) tasks across in-flight changes are marked open although the code they describe is present and verified in the working tree (SQLC billing-provider queries, `test:e2e` script, GitLab E2E job), keeping those changes in in-progress limbo. Two E2E specs additionally exhibit pre-existing flaky/hang failures under parallel load, blocking a green full-suite run.

## What Changes

- **Provider-aware billing UI**: `app/dashboard/settings/components/subscription-tab.tsx` branches the cancel flow by provider — when `isMercadoPagoEnabled()`, cancellation calls the existing `cancelMPSubscription` server action, which enforces a Stytch session and `canManageSubscriptions` permission server-side and posts to `POST /api/subscriptions/mp-cancel`; otherwise the existing Polar `cancelSubscription` flow is kept. Provider-appropriate copy replaces the "Billed via Polar" fallback (line 197).
- **Provider-aware paywall**: `components/billing/subscription-paywall.tsx` renders provider-appropriate inactive-state copy (MercadoPago vs Polar) based on `isMercadoPagoEnabled()`.
- **Task-state reconciliation**: mark done — `add-mercadopago-billing` 3.2/3.3/3.4 (SQLC queries exist at `internal/db/postgres/sqlc/gen/organizations.sql.go:372,633`), `add-crm-e2e-tests` 2.3 (`test:e2e` exists in `package.json:9`) and 11.1 (GitLab CI `run-frontend`/`run-e2e` jobs exist in `go-b2b-starter/.gitlab-ci.yml:22,41`).
- **E2E flaky-spec hardening**: `e2e/specs/whatsapp-inbox.spec.ts:65` (fetch stall on duplicate-delivery idempotency) and `e2e/specs/deals.spec.ts:91` (`waitForResponse` on `POST /api/crm/negocios` timing out under parallel load). Code fixes land here; full-suite execution verification is deferred (environment), not blocked.
- **Archive-decision records**: `fix-migration-renumber` 3.4 and `add-whatsapp-embedded-signup` 7.3 record explicit archive-deferral entries in their own tasks.md.

## Capabilities

### New Capabilities
- `billing-provider-ux`: provider-aware billing surfaces — subscription cancellation branches by active provider (Polar default, MercadoPago when enabled), and the paywall inactive state renders provider-appropriate messaging.

### Modified Capabilities
<!-- None: settings-ui spec covers member/module/playbook/profile requirements, not subscription surfaces; MP billing behavior lives in in-flight deltas (billing-provider-routing, mercadopago-checkout), not in a living spec. -->

## Impact

- **Frontend**: `subscription-tab.tsx` (cancel branch + provider copy), `subscription-paywall.tsx` (provider copy), `e2e/specs/whatsapp-inbox.spec.ts`, `e2e/specs/deals.spec.ts`. No new dependencies.
- **Backend**: none — `POST /api/subscriptions/mp-cancel` already registered (`internal/modules/billing/routes.go:28`); SQLC queries already generated.
- **Database**: no migrations.
- **CI**: no config changes; GitLab E2E job already present.
- **Auth boundary**: no Stytch B2B contract changes; the MP cancel server action already verifies the Stytch session JWT and RBAC permission before the outbound call (mirrors the existing action pattern; no local credential storage involved).
- **Rollback strategy**: Git — revert the change commits (UI-only, additive; reconciliation edits revert with the change). DB — none needed (no migrations). Stytch tenant policy state — unaffected (no RBAC/auth policy changes), so no Stytch-side rollback is required.

## Non-Goals

- No backend changes, no new endpoints, no migrations, no local credential storage.
- No live-sandbox verification (MercadoPago/Polar webhooks, checkout e2e, Siigo, embedded-signup smoke, Stytch scope e2e) — remains deferred to deployment as already recorded in the owning changes.
- No CI live run (no GitHub credentials in this environment).
- No autopilot, multi-model routing, prompt registry, semantic cache, or evals (post-MVP; see `ROADMAP.md`).

## Assumptions

- MercadoPago enablement is signaled by the presence of `NEXT_PUBLIC_MERCADOPAGO_PLAN_ID` via `isMercadoPagoEnabled()`; the UI does not query a backend provider endpoint for cancel/copy branching.
- The two flaky E2E specs are fixable by response-wait/retry hardening without a running stack to verify; full-suite verification happens in CI/deployment.
