# Proposal: reconcile-spec-sync-gaps

## Why

A gap analysis of archived changes vs. living specs found two archived changes whose delta specs were never folded into `openspec/specs/`: `add-e2e-edge-coverage` (6 requirements describing E2E coverage that exists in the Playwright suite but is unspecified) and `fix-frontend-build` (2 requirements describing `ApiClient` query-param and query-key behavior that the code implements but the spec is silent on). Additionally, the `GET /api/auth/check-email` endpoint is live in the Go backend and consumed by the auth page, yet no living capability documents its contract. The behavioural source of truth is therefore incomplete: code and tests implement behavior the specs do not record.

## What Changes

- Fold the 6 unfurled `add-e2e-edge-coverage` requirements into `crm-test-infrastructure` (5) and `crm-whatsapp-e2e` (1) as living spec requirements. Premise-verified against the current Playwright suite (`surrounding-processes.spec.ts`, `whatsapp-edge-cases.spec.ts`, `inbox-ui.spec.ts`).
- Fold the 2 unfurled `fix-frontend-build` requirements into `frontend-api-client`: query-parameter serialization (`params` + `URLSearchParams`) and query-key namespaces (`queryKeys.whatsappConfig`). Premise-verified against `lib/api/api/client/api-client.ts` and `lib/hooks/queries/query-keys.ts`.
- Add a new `auth-email-check` capability documenting the `GET /api/auth/check-email` contract (200 exists / 404 not found / 400 invalid / 500 error) as implemented by `MemberHandler.CheckEmail` — a read-only local-DB email existence check (no Stytch API call, no credential material).
- Fix the auth page's API base URL resolution: `app/auth/page.tsx` currently falls back to a hardcoded cross-origin `http://localhost:8080/api`, contradicting the `frontend-api-client` same-origin default rule (origin of the archived `fix-auth-cross-origin-csp` fix). The check-email fetch SHALL resolve the base URL through the shared ApiClient configuration (same-origin relative `/api` default).
- **No code changes** for the E2E and ApiClient folds — the behavior and tests already exist; this change only records them in the authoritative specs.

## Capabilities

### New Capabilities
- `auth-email-check`: contract of `GET /api/auth/check-email` — org-agnostic email existence check (200/404/400/500), read-only local DB lookup, no session required, and same-origin base URL usage by the auth page (no hardcoded cross-origin fallback).

### Modified Capabilities
- `crm-test-infrastructure`: add 5 requirements — cross-organization data isolation, pagination, outbound reply persistence, mock-auth guard, and RBAC boundary are E2E-tested.
- `crm-whatsapp-e2e`: add 1 requirement — WhatsApp webhook edge-case scenarios are E2E-tested.
- `frontend-api-client`: add 2 requirements — the ApiClient supports query-parameter serialization (`params` option, `URLSearchParams`, `&`-joining) and exposes the `queryKeys.whatsappConfig` namespace (`all`, `detail()`).

## Impact

- **OpenSpec**: delta specs for the four capabilities above; living specs gain 8 requirements and one new capability.
- **Frontend**: one small change to `app/auth/page.tsx` (base URL resolution via shared ApiClient config; behavior for 200/404/400 unchanged). No other FE changes.
- **Backend**: none — the `check-email` handler, service, and repository are unchanged; the fold records existing behavior.
- **Stytch B2B**: no API contract or tenant policy changes. The `check-email` endpoint performs a local DB read (`organizations` table via `localOrgRepo.GetByUserEmail`) and never calls Stytch APIs; sessions and RBAC remain exclusively Stytch-owned per the Dual SSOT constitution.
- **Rollback (Git)**: revert the auth-page base-URL edit (`git checkout` on `app/auth/page.tsx`); delete the new `auth-email-check` spec and the added requirements via `git revert` of the spec commit.
- **Rollback (Stytch)**: none required — no Stytch tenant state is touched.

## Non-Goals

- NOT introducing local credential storage: no passwords, MFA tokens, or session tokens are stored or read in this change; the `check-email` lookup reads only membership/email existence from the local organizations table (which stores `stytch_member_id` / `stytch_organization_id` foreign keys only).
- NOT changing the check-email message semantics (the current distinct 404 message is an enumeration signal; enforcing anti-enumeration per `stytch-nextjs-components` is a product decision deferred to a separate change — recorded in design.md).
- NOT folding the abandoned `password-auth` deltas (explicitly rejected by `abandon-password-auth`).
- NOT folding the `mvp-launch` governance deltas (one-time launch meta-process; durable parts already generalized into living `governance-workflow` / `ci-pipeline` requirements).
- NOT re-archiving or re-verifying the E2E suite itself; only recording its existing coverage contractually.
