## Why

The admin panel is half-built but half-hidden. Core product features exist as routes and components yet are unreachable: CRM (contacts, companies, deals, tags, pipelines, activity) and Inbox have no sidebar entry, the WhatsApp Business connection UI (`WhatsAppConfigSection`) is orphaned — mounted nowhere — and workspace settings are read-only: org name cannot be edited and member roles freeze at invite time. Admin UX falls short of the 2026 SaaS baseline not because features are missing, but because they are invisible or unmanageable.

## What Changes

- **Sidebar navigation** — add "CRM" (`/dashboard/crm`) and "Inbox" (`/dashboard/inbox`) entries to the dashboard sidebar, visible to users whose entitlements/permissions permit them, matching the existing permission-filtered nav pattern.
- **WhatsApp settings view** — mount the existing `WhatsAppConfigSection` (embedded signup + config form + active toggle) as a real settings view reachable from the overview, satisfying the existing `whatsapp-config-frontend` spec: "Messaging" card in overview for `org:manage` users, `?view=whatsapp` navigation, permission gating, loading/error states.
- **Editable workspace name** — workspace card in settings gains an inline edit form persisting via `PUT /organizations` (`org:manage`), passing through the required `status` field.
- **Member role management** — member list row gains a role control persisting via `PUT /auth/members/:member_id/role` (`org:manage`), aligning the role vocabulary used by the frontend with the roles accepted by the backend (`admin`, `approver`, `member`).
- **Dead header actions** — wire the inert Support and Preferences header buttons or remove them.
- **Audit log view** — a read-only audit view in settings listing the organization's unified CRM activity (notes, calls, emails, meetings, tasks, WhatsApp messages, system events) from the existing `GET /api/crm/actividades` endpoint, with a tipo filter and pagination. Gated in the frontend by the `audit:view` permission; the backend route continues to enforce `contact:view`.

Minimal backend changes: `PUT /organizations` (`UpdateOrganization`) is extended to sync the display name to Stytch B2B (`Organizations.Update`) before the local write; a new `PUT /auth/members/:member_id/role` endpoint (`ChangeMemberRole`) syncs the member role to Stytch (`Members.Update`) before the local write and enforces a last-admin guard. No migrations or SQLC model changes. No local credential storage introduced; identity and roles continue to live in Stytch B2B.

## Capabilities

### New Capabilities

- `admin-panel-navigation`: Dashboard sidebar SHALL expose Inbox and CRM to entitled users; settings overview SHALL surface a WhatsApp section card.
- `workspace-settings-management`: Admin SHALL be able to edit workspace display name and change a member's role from the settings UI, both persisted to Stytch-backed backend endpoints.
- `admin-panel-audit-log`: Settings SHALL expose a read-only audit log view listing unified organization CRM activity with tipo filtering and pagination, visible only to users holding the `audit:view` permission.

### Modified Capabilities

- `whatsapp-config-frontend`: Extend to require the embedded signup connect path (`Connect WhatsApp` flow) as the primary no-config entry point from the settings view, in addition to the existing manual-config form.

## Impact

- **Frontend** (`next_b2b_starter/`): `components/layout/sidebar.tsx` (Inbox/CRM entries with entitlement gating), `app/dashboard/settings/components/settings-content.tsx`, `profile-section.tsx` (workspace name edit with org-status passthrough), `member-list.tsx` (role control), new `app/dashboard/settings/components/audit-log-view.tsx`, new `lib/api/api/repositories/*` methods for org update and member role update. No new dependencies.
- **Backend** (`go-b2b-starter/`): extend `internal/modules/organizations` — `UpdateOrganization` calls Stytch `Organizations.Update` (display_name) before the local write; new `PUT /auth/members/:member_id/role` (`memberService.ChangeMemberRole`, gated `org:manage`) calls Stytch `Members.Update` (role slug) before the local write and enforces a last-admin demotion guard. Existing endpoints consumed unchanged: `PUT /organizations` (gated `org:manage`), `GET /api/crm/actividades` (gated `contact:view`), verified in `internal/modules/organizations/routes.go` and `internal/modules/crm/routes.go`. Outbound Stytch calls are guarded by a circuit breaker on the shared Stytch client (`internal/platform/stytch`).
- **Specs**: new `admin-panel-navigation`, `workspace-settings-management`, `admin-panel-audit-log`; delta to `whatsapp-config-frontend`.

## Non-Goals

- No billing invoice history, or payment-method management (deferred).
- No unified merge of `agent_actions` into the audit view — the audit log shows CRM activity only; agent governance rows remain scoped to the agent module.
- No WhatsApp template catalog or quick-reply management.
- No DB migrations or SQLC model changes (the audit log consumes the existing `GET /api/crm/actividades` endpoint unchanged; the only new backend surface is the single member-role endpoint `PUT /auth/members/:member_id/role`).
- No personal profile editing (display name/email remain managed by Stytch B2B).
- No local storage of passwords, MFA tokens, session tokens, or Stytch credentials.

## Rollback

- **Git state**: revert the frontend commits; all changes are additive (new nav items, new settings view wiring, new repository methods) so revert is clean and removes no data.
- **Stytch tenant policy state**: member role changes are applied through `PUT /auth/members/:member_id/role`; rollback re-issues the previous role via the same endpoint per affected member. No Stytch org/policy mutations are performed by this change beyond the member role updates exposed by that endpoint.
