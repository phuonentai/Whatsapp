# Auth Event Audit Trail

## Why

The settings audit log (`?view=audit`, permission `audit:view`) currently shows only CRM activity (notes, calls, emails, tasks, WhatsApp/system events). Sign-in events — magic-link requests, successful/failed logins, MFA challenges, logouts — are not recorded anywhere. Modern SaaS security reviews (and the product's own `STYTCH_CONFIGURATION.md` "Monitor failed attempts" recommendation) expect an auth-event trail for incident response, account-takeover detection, and compliance.

## What Changes

- Record auth events into the existing activity/audit stream (CRM activity table, `tipo=sistema`), so they appear in the current `?view=audit` UI with no new surface.
- Events recorded: magic link requested (per org), login succeeded, login failed (token consumption rejected), logout, and (once MFA lands) MFA challenge passed/failed.
- Events are written by the Next.js server actions (`sendMagicLink`, `consumeMagicLink`, `logout`) via the authenticated Go API (existing activity-create path), using `stytch_member_id` for attribution. No session tokens, JWTs, magic-link tokens, or passwords are stored — only event type, member id, org id, timestamp, and a non-sensitive detail (e.g., failure reason code from Stytch).
- The recording path is best-effort: an audit-write failure MUST NOT block or fail the auth action (non-critical path, log warning).

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `admin-panel-audit-log`: the audit stream SHALL include auth events (login success/failure, magic-link request, logout) in addition to CRM activity, visible under the existing `?view=audit`.

## Impact

- **Frontend:** `sendMagicLink.ts`, `consumeMagicLink.ts`, `logout.ts` call the audit-record helper after the auth outcome is known.
- **Backend:** the activity-create API already exists (`POST /api/crm/actividades`); no schema change required. If a dedicated lightweight endpoint is preferred, add `POST /api/auth/events` — design decision below.
- **Dependencies:** none new.
- **Stytch:** no tenant policy changes; event data derives from Stytch API responses already received by the actions.

## Rollback

- **Git:** revert the action changes; audit calls are additive and non-blocking.
- **Stytch tenant policy state:** none modified.

## Non-Goals

- NOT storing session tokens, JWTs, magic-link tokens, IP addresses of members, or other sensitive auth material in the audit stream (SSOT: sessions live in Stytch; audit stores only attribution + event type).
- NOT building a separate auth-audit UI or endpoint surface; reuses the existing audit view.
- NOT capturing full HTTP request/response bodies.
