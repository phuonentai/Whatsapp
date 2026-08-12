# Auth Event Audit Trail — Design

## Context

Auth outcomes are already known inside three Next.js server actions:
- `sendMagicLink` (`lib/actions/auth/send-magic-link.ts`) — knows email, member orgs, whether send happened.
- `consumeMagicLink` (`lib/actions/auth/consume-magic-link.ts`) — knows success/failure, `mfa_required`, member/org.
- `logout` (`lib/actions/auth/logout.ts`) — knows member/org (from session) and revoke outcome.

The audit UI reads `GET /api/crm/actividades` (tipo filterable); the create path is `POST /api/crm/actividades` with `tipo=sistema`, authenticated via the Go API. CRM activity rows carry `organization_id`, performing member, timestamp, subject/content, linked entity.

## Decisions

### D1 — Reuse the CRM activity API (no new table)

- New helper `recordAuthAudit({ type, orgId, memberId, detail })` in `lib/auth/audit.ts` (server-only) that POSTs an activity row (tipo=sistema, subject = auth event label, content = compact JSON of `{type, detail}`) to the existing Go endpoint using the member's session JWT.
- No new DB schema; no new backend endpoint. Rationale: smallest diff, appears in the existing `?view=audit` UI immediately, reuses existing RBAC (`audit:view` gating is on the read side).

### D2 — Best-effort, non-blocking

- Wrap the audit POST in try/catch; on failure log `console.warn` and continue. The auth action's own result is returned regardless (state-transition invariant: audit failure never changes auth outcome).
- Never pass tokens/JWTs to the audit payload; only `{type, member_id, organization_id, detail}` where `detail` is a bounded enum string (e.g., `"invalid_token"`, `"expired_token"`, `"mfa_required"`) — never raw Stytch error bodies.

### D3 — Event taxonomy (enum)

`magic_link_requested`, `login_succeeded`, `login_failed`, `logout`, `mfa_challenge_passed`, `mfa_challenge_failed` (the last two land with the MFA change; the enum is defined now so the MFA change can reuse it).

### D4 — Multi-org note

For `sendMagicLink`, if the member belongs to N orgs and N links are sent, record one row per org (N rows) — matches how the send loops per org.

## Stytch Boundary

- All data recorded is derived from Stytch API responses the actions already receive; no new outbound Stytch calls are introduced by this change.
- Circuit-breaker/fallback: audit recording has no Stytch dependency (Go API only); a Stytch outage at login already yields `login_failed` from existing error paths, which is exactly what we want recorded.

## Security Invariants

- Audit rows MUST NOT contain session tokens, JWTs, magic-link tokens, or member email/IP in `detail` (SSOT compliance; attribution is `stytch_member_id`).
- The Go activity endpoint already enforces org isolation (`RequestContext.OrganizationID` from the session) — the helper must pass the authenticated member's org, never client-supplied orgs.

## Testing Strategy

- Unit: `recordAuthAudit` posts correct payload shape; swallows API errors (non-blocking).
- Integration: mocked Go activity endpoint receives one row per event type; failure injection proves auth outcome unaffected.
- Existing `admin-panel-audit-log` scenarios remain green (read side unchanged; new rows are tipo=sistema, already rendered).
