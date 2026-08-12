# Stytch Ops & Security Suite

## Why

The free-tier gap analysis surfaced four Stytch B2B capabilities that are free (verified: user impersonation on all plans; device fingerprinting 10K/mo; webhooks; event-log streaming in free beta) and that the platform does not consume. They close real gaps: support agents cannot act on behalf of members (impersonation); the governance audit stream only records *auth* events (`admin-panel-audit-log` — `login_succeeded`, `magic_link_requested`, …) and nothing from the member/org lifecycle (webhooks); the custom rate limiters are single-instance and blind to device-level risk (fingerprinting); and auth observability ends at the app (event streaming). All four are enablement + thin-integration work against the existing constitution invariants (breaker-guarded outbound calls, HMAC-verified webhooks per AGENTS.md, no local credential storage, read-only fallback).

## What Changes

- **Support impersonation**: admins with `org:manage` + a support capability can mint an impersonation token for a member in their org (token creation via the Stytch dashboard support console; exchange via `POST /v1/b2b/impersonation/authenticate`). The impersonated session is **60 minutes, non-extendable**, carries impersonation claims in the JWT (`impersonating: true`, `impersonator_id`/`impersonator_email_address`), and is surfaced in the UI as "viewing as" with an exit action. Every impersonation session mint/enter/exit records a `support_impersonation_started` / `support_impersonation_ended` audit row (`tipo=sistema`).
- **Stytch lifecycle webhooks**: a Svix-style webhook ingress endpoint (`/api/webhooks/stytch`) with `Webhook-Id` / `Webhook-Timestamp` / `Webhook-Signature` HMAC-SHA256 verification (`whsec_` secret, per AGENTS.md "webhooks are verified via signature/HMAC headers before any DB mutation"), idempotent on `event_id` (transaction-isolated dedup), consuming member/org lifecycle events (`direct.member.create/update/delete`, `direct.organization.update`, `dashboard.member.*`, `scim.member.*`). Verified events append **governance audit rows** (member invited, role changed, member removed, org updated, SCIM deprovisioned) to the existing audit stream — extending `admin-panel-audit-log` from auth-only to auth + governance. Payloads are treated as a trigger to re-fetch fresh state via the Stytch API (event-ordering-safe), never as authoritative state.
- **Device fingerprinting**: the client captures a Stytch device fingerprint (`stytch.b2b.fingerprint()` from the pinned `@stytch/vanilla-js`) at login; the `device_id` is sent to the backend and stored per member-org (no PII), within the **10K fingerprints/mo** free budget. The stored fingerprint is exposed to the existing risk/rate-limiting surfaces as an optional per-device dimension (limiter behavior unchanged in this change) and displayed on the member profile for support triage.
- **Event log streaming**: Stytch event-log streaming (free beta) destinations configured for **Datadog** (site + API key) and/or **Grafana Loki** (public `/loki/api/v1/push`, gzipped JSON) via dashboard config; documented in `STYTCH_CONFIGURATION.md` with the auth-event taxonomy. No application code.
- **Docs**: `STYTCH_CONFIGURATION.md` updated (impersonation enablement + support console usage, webhook endpoint/secret rotation, fingerprint budget, streaming destinations).

## Capabilities

### New Capabilities
- `support-impersonation`: impersonation token exchange, impersonated session handling (60-min non-extendable, JWT impersonation claims), "viewing as" UI with exit, impersonation audit events, `org:manage` + support-capability gating.
- `stytch-lifecycle-webhooks`: Svix webhook ingress, HMAC verification, idempotent processing, governance audit feed from member/org lifecycle events.
- `device-fingerprinting`: client fingerprint capture, per-member-org storage, 10K/mo budget guardrail, risk-surface exposure.
- `event-log-streaming`: Datadog/Grafana Loki streaming destinations (dashboard config) + documentation.

### Modified Capabilities
- `admin-panel-audit-log`: audit stream extended from auth events to include governance events sourced from verified Stytch lifecycle webhooks (member invited/role changed/removed, org updated, SCIM deprovisioned), same `tipo=sistema` stream and read-only view.

## Impact

- **Backend**: new `internal/modules/stytchwebhooks` (or `platform/stytch/webhooks`) — Svix signature verifier, `POST /api/webhooks/stytch` handler, `event_id` dedup store (new small migration), governance-event mapper writing `tipo=sistema` audit/activity rows; impersonation: support endpoint(s) to exchange an impersonation token via `POST /v1/b2b/impersonation/authenticate` behind the existing breaker + impersonation audit rows; fingerprint: `POST /api/risk/device-fingerprint` (authenticated) persisting `device_id` per member-org.
- **Database**: one small migration (webhook `event_id` dedup table + optional member device fingerprint column or table); no credential material anywhere.
- **Frontend**: member profile shows "Impersonate" (support-capability gate) + "viewing as" banner with exit; login surface calls `stytch.b2b.fingerprint()` and posts the `device_id`; audit view unchanged (new event types appear automatically).
- **Dependencies**: none new — Svix verification implemented with stdlib HMAC (pattern exists for Meta `X-Hub-Signature-256` and MercadoPago signatures); `@stytch/vanilla-js` already pinned.
- **Stytch tenant policy state**: webhook endpoint registered per project (test + prod), impersonation enabled, streaming destinations configured — all reversible in the dashboard.
- **Pricing posture**: all free within limits — impersonation included on all plans; fingerprints 10K/mo ($0.005 each after); streaming free beta. **Not** the paid bot-detection/audit-logs products.

## Rollback

- **Git**: revert the change (webhook module + migration, impersonation endpoints, fingerprint capture, docs). Migration has a down file.
- **Stytch tenant policy state**: webhook destination deleted in dashboard (or signature secret rotated); impersonation disabled per project; streaming destinations disabled. No org-policy change is required; impersonation sessions are 60-min non-extendable and expire automatically.

## Non-Goals

- **NO local credential storage**: webhook secrets (`whsec_`) live in env/secrets management, not the DB; impersonation tokens and sessions are never stored locally; fingerprints contain no PII and no device-identifying material beyond the Stytch `device_id`.
- NOT the paid fraud products (bot detection, CAPTCHA, intelligent rate limiting) — fingerprints are a free additive risk signal only; custom in-process limiters remain the enforcement mechanism.
- NOT Stytch Audit Logs ($799/mo) — the project's own `tipo=sistema` audit stream remains; webhooks only feed it.
- NOT org-creation, JIT, or auth-method changes (see `stytch-enterprise-suite`).
- NOT M2M service auth (tracked in `stytch-m2m-service-auth`).
- Impersonation is member-level within the impersonator's own org only — cross-org support impersonation and dashboard-admin (Stytch-dashboard-user) impersonation are out of scope.
