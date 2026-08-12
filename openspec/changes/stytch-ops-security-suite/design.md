# Stytch Ops & Security Suite — Design

## Context

Free-tier capabilities verified against Stytch docs/pricing (2025): **user impersonation** is on all plans; **device fingerprinting** is 10K fingerprints/mo free ($0.005 each after); **webhooks** are free (Svix-based, standard webhook signing); **event log streaming** is a free beta (Datadog, Grafana Loki). The paid fraud/audit products (bot detection, CAPTCHA, intelligent rate limiting, Audit Logs $799/mo) are out of scope. Existing surfaces this change plugs into: `admin-panel-audit-log` records auth events as `tipo=sistema` CRM activity rows visible in `?view=audit` (best-effort, non-blocking, no credential material — verified requirements); `sign-in-rate-limiting` uses in-process sliding-window limiters (single-instance documented); webhook HMAC patterns already exist for Meta (`X-Hub-Signature-256`) and MercadoPago; the Stytch adapter breaker (`Client.Run`, threshold 5/10s/2) guards outbound Stytch calls; `@stytch/vanilla-js` (B2B) is pinned and already used.

**Verified Stytch contracts:**

- Impersonation: `POST /v1/b2b/impersonation/authenticate` authenticates an impersonation token and returns a **60-minute, non-extendable** session (`session_token`/`session_jwt`) plus `impersonator_id`/`impersonator_email_address`; the JWT carries impersonation claims. Token minting happens from the Stytch dashboard support console (an admin with impersonation permission); an API token-create variant may exist — verify in the test project (recorded in tasks).
- Webhooks: Svix standard webhooks — headers `Webhook-Id`, `Webhook-Timestamp`, `Webhook-Signature`; signing secret `whsec_…`; HMAC-SHA256 over `{id}.{timestamp}.{payload}`; events use `source.object_type.action` naming (`direct.member.create/update/delete`, `direct.organization.create/update`, `dashboard.member.*`, `scim.member.*`); recommended practice is to treat the payload as a trigger and re-fetch authoritative state via the API.
- Fingerprinting: `stytch.b2b.fingerprint()` returns a `device_id`; free 10K/mo.
- Streaming: dashboard-configured destinations — Datadog (site + API key) or Grafana Loki (public `/loki/api/v1/push`, gzipped JSON); free beta.

## Goals / Non-Goals

**Goals:**

- Support agents can act as a member of their own org (impersonation) with a visible, time-bounded, fully audited session.
- Member/org lifecycle events flow into the existing governance audit stream via verified, idempotent Stytch webhooks.
- Device-level risk signal (free fingerprints) captured without changing limiter behavior.
- Auth observability exported to Datadog/Grafana with zero application code.
- Strict SSOT: no local credential material; webhooks never mutate before verification; audit rows stay credential-free.

**Non-Goals:**

- Paid fraud/audit products; M2M; org-creation/JIT/auth-method changes; cross-org impersonation; Stytch-dashboard-user impersonation; limiter behavior changes.

## Decisions

### D1 — Impersonation: dashboard-minted token, Go exchange, support-capability gate

Flow: a support member with `org:manage` (frontend gate) opens a member profile → the support console action creates an impersonation token (dashboard-side; or API variant if verified) → the frontend sends the token to a Go endpoint → Go exchanges it via `POST /v1/b2b/impersonation/authenticate` behind the existing breaker (breaker-open → 503 structured error `impersonation_unavailable`) → the returned `session_jwt` is set with the same cookie config as normal login, and the UI renders a persistent "viewing as <member>" banner with an exit action (`sessions.revoke` + redirect).

- **JWT claims**: the impersonated session JWT carries `impersonating: true` + `impersonator_id`/`impersonator_email_address`; the Go auth middleware reads these claims to (a) log impersonated traffic, (b) block sensitive surfaces if desired later, (c) reject if the impersonator's org ≠ target member's org (defense in depth — Stytch enforces at mint).
- **60-min non-extendable** — no session extension; exit = revoke; session expiry is enforced by Stytch + JWKS verification.
- **Audit**: `support_impersonation_started` (member, org, impersonator) / `support_impersonation_ended` rows, best-effort per `admin-panel-audit-log`.
- **Rationale**: dashboard minting means no custom token-CRUD + permission system; the Go exchange keeps the frontend free of direct secret handling and routes through the existing breaker. Rejected: client-side exchange (`impersonation.authenticate` in vanilla-js) — bypasses the breaker seam and audit hook.

### D2 — Webhooks: Svix ingress, verify-then-dedup-then-fetch, governance mapper

`POST /api/webhooks/stytch` (public, no session) — first **verify** the Svix signature (`whsec_` secret from env, HMAC-SHA256 over `{Webhook-Id}.{Webhook-Timestamp}.{body}`, replay window on `Webhook-Timestamp`, constant-time compare; reject with 401 before any work). Then **idempotency**: `event_id` (Webhook-Id) checked against a dedup table inside the same transaction as the resulting writes (mirrors the `durable-message-pipeline` outbox pattern; replays return 200). Then **map + re-fetch**: for member/org lifecycle events, re-fetch fresh state via the Stytch API (breaker-guarded) and append governance audit rows (`member_invited`, `member_role_changed`, `member_removed`, `organization_updated`, `member_deprovisioned`) to the `tipo=sistema` stream with bounded detail (member id, org id, role before/after, timestamp). Unknown/unmappable events are logged and acked.

- **Rationale**: Svix standard-webhooks is what Stytch ships — using it verbatim avoids inventing a signature scheme; verify-before-any-DB-mutation is the AGENTS.md constitution invariant; fetch-don't-trust payload handles event ordering (verified Stytch guidance).
- Auth events (`login_succeeded` etc.) remain app-recorded as today — webhooks cover only the lifecycle events the app cannot observe.

### D3 — Fingerprinting: client capture → authenticated store → risk surface (no limiter change)

The login surface (and `stytch-login-surface` pre-built flow) calls `stytch.b2b.fingerprint()` once per session start; the resulting `device_id` is POSTed to `POST /api/risk/device-fingerprint` (authenticated, same-origin) and stored per member-org (one column/table). The stored fingerprint is (a) shown on the member profile for support triage, (b) exposed to `sign-in-rate-limiting` as an **optional** per-device dimension in a follow-up (limiter behavior unchanged now), (c) bounded by the 10K/mo budget — a per-project counter/log warns when approaching the limit. Fingerprints contain no PII; only the opaque Stytch `device_id` is stored.

### D4 — Event streaming: dashboard-only, documented

No application code. Configure Datadog (site + API key) and/or Grafana Loki endpoints in the Stytch dashboard; document the destinations, the auth-event taxonomy, and rotation/disable steps in `STYTCH_CONFIGURATION.md`. Note the free-beta status and the 30-day dashboard event-log retention.

### D5 — Gating

Impersonation entry: `org:manage` + support capability (reuse the existing permission model — a `support:impersonate` permission added to the Stytch RBAC policy `admin` role per `stytch-authorization` policy-driven rules). Webhook endpoint: no auth (signature is the auth). Fingerprint POST: any authenticated member (self-service).

## Risks / Trade-offs

- **Impersonation abuse** → org-scoped only, `org:manage` + explicit support permission, 60-min non-extendable, impersonation claims in every JWT, always-on "viewing as" banner, both start/end audited. Risk accepted and visible.
- **Webhook replays / out-of-order delivery** → `event_id` dedup in-transaction; fetch-fresh-state instead of trusting payload; replay window on `Webhook-Timestamp`.
- **Webhook secret leakage** → `whsec_` in env/secrets management only, rotation documented, never in DB or logs.
- **Fingerprint budget overrun (10K/mo)** → counter + warning log; beyond budget costs $0.005/fingerprint — acceptable at projected MAU; no hard cap gate.
- **B2B impersonation token minting path** (dashboard-only vs API) undocumented in our SDK types → assumption recorded; verified in the Stytch test project during E2E.
- **Webhook event-name drift** (`direct.*`/`dashboard.*`/`scim.*` taxonomy) → mapper logs-and-acks unknown events; event names verified in test project E2E before prod.
- **Audit row proliferation** from lifecycle events → bounded detail fields (no payload dumps), same best-effort/non-blocking contract.

## Migration Plan

1. Test project: enable impersonation + register webhook endpoint (test URL) + configure streaming destinations; verify signature verification with the Svix test tools.
2. Backend: webhook ingress (verify → dedup → fetch → map) with migration; impersonation exchange endpoint; fingerprint endpoint. All behind existing invariants.
3. Frontend: impersonate entry + "viewing as" banner; fingerprint capture at login.
4. Prod: mirror config; rotate webhook secret before go-live; monitor dedup/fingerprint counters.
5. Rollback: Git revert + down migration; dashboard: delete webhook endpoint, disable impersonation, disable streaming.

## Open Questions

- B2B impersonation token minting: dashboard support console only, or is there an API path we should expose for our own support tooling? (Verified in test project; recorded in tasks.)
- Which event names does the test project actually emit for SCIM deprovisioning (`scim.member.delete` vs `scim.member.update` with status)? (Verified during E2E.)
- Should the fingerprint device_id become a rate-limiter dimension now or in a follow-up? (Default: follow-up; this change only captures + stores.)
- Streaming: Datadog, Grafana Loki, or both? (Config decision; docs cover both.)
