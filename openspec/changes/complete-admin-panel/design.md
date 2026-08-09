## Context

The dashboard sidebar only exposes Dashboard, Knowledge Base, and Settings (`components/layout/sidebar.tsx`), yet Inbox (`/dashboard/inbox`) and CRM (`/dashboard/crm`) are fully built. `WhatsAppConfigSection` (embedded signup + config form + toggle) exists but is mounted nowhere. The settings page renders profile/workspace info read-only and freezes member roles at invite time.

Premise verification surfaced a dual-SSOT constraint: `profile.organizationName` and member roles are sourced from Stytch B2B (`member_service_impl.go` builds `ProfileOrganization.Name` from `authOrgRepo.GetOrganization`), but the existing update endpoints — `PUT /organizations` (`UpdateOrganization`) and `PUT /accounts/:id` (`UpdateAccount`) — write only the local PostgreSQL tables and never call Stytch. Backend role vocabulary is `admin|approver|member` (`UpdateAccountRequest` binding); the frontend models `admin|manager|member` and falls back to member config for `approver`, mislabeling approvers today.

## Goals / Non-Goals

**Goals:**
- Make Inbox, CRM, and WhatsApp connection reachable from the dashboard/settings UI.
- Allow admins to edit the workspace display name and change member roles without creating local-vs-Stytch identity drift.
- Remove inert header controls.

**Non-Goals:**
- Billing history, payment methods, WhatsApp template catalog.
- Local credential storage of any kind (passwords/MFA/session tokens remain Stytch-managed).
- Rewriting the WhatsApp embedded signup flow.
- Merging agent governance rows (`agent_actions`) into the audit view — MVP scope is CRM activity only.

## Decisions

### 1. Sidebar navigation — static entries with entitlement gate
Add `Inbox` and `CRM` entries to `mainNavigation` in `sidebar.tsx` (entries are already present in the working tree). Gate visibility via the existing `anyPermissions` filter where a permission maps; where CRM/Inbox are entitlement-gated (module-based), gate with `useModule`/entitlement helper instead of role permission. Labels stay English for nav consistency (Dashboard, Inbox, CRM, Knowledge Base, Settings).

- **Implemented:** entries render for all users; entitlement gating (hide when the corresponding `funcionalidades`/`modulos` flag is off) is layered on via `useEntitlementQuery` in the client component, showing entries while entitlements load to avoid nav flicker. Spec scenario "user without entitlement does not see restricted entries" is enforced by this gate.

- **Alternative rejected:** auto-redirecting `/dashboard` to feature picker — more surface, less value now.

### 2. WhatsApp settings view — mount existing component
Add `whatsapp` to `SettingsView` union, `parseViewParam`, `DETAIL_META`, an overview card ("Messaging"), and `renderDetailContent` branch in `settings-content.tsx`, gated by `ORG_MANAGE`. Reuse `WhatsAppConfigSection` unchanged — it already implements the embedded signup flow, polling, manual form, and toggle required by the `whatsapp-config-frontend` spec. This satisfies the existing spec's overview-card and `?view=whatsapp` requirements plus the new embedded-signup delta.

### 3. Workspace display-name edit — extend `UpdateOrganization` to sync Stytch
`PUT /organizations` keeps its route and payload shape but the service SHALL additionally call the Stytch B2B `Organizations.Update` API (`display_name`) before/within the same flow, then persist locally, so both SSOTs stay in phase. On Stytch failure, reject the request (do NOT write local-only) and surface an error to the admin. FE: add an edit form on the Workspace card calling a new repository method that passes `name` and the current `status` (the binding requires `status`).

- **Implemented:** `UpdateOrganization` syncs `display_name` to Stytch via `stytchOrganizationRepository.UpdateOrganization` (`Organizations.Update`, stytch-go v18) before the local write and rejects on failure (`organization_service.go`). FE form sends `name` + `organizationStatus` passed through from the profile payload.
- **Rationale:** writing local-only would recreate the drift the constitution forbids (failure mode 1: schema-identity drift).
- **Alternative rejected:** editing only the local row — produces a UI that shows a name Stytch still owns.

### 4. Member role change — new Stytch-synced endpoint
The member list exposes Stytch `member_id`, but `PUT /accounts/:id` addresses accounts by local integer id, so it cannot be called from the member UI. Add `PUT /auth/members/:member_id/role` (org:manage) implemented in `memberService.ChangeMemberRole`: it SHALL call Stytch B2B `Members.Update` (role slug) FIRST and reject the request on failure (no local-only write), enforce the last-admin guard, then persist the local row. Role vocabulary is `admin|approver|member`, matching `mapRoleSlugToAccountRole` and the account binding. FE: role control on member rows (disabled for current user and pending members), aligned `MemberRole`/`getRoleConfig`/`DEFAULT_ROLES` from `admin|manager|member` to `admin|approver|member`, fixing the existing approver mislabel.

- **Implemented:** `UpdateAccount` assigns the role slug through `stytchMemberRepository.AssignRoles` (`Members.Update`) before the local write and rejects on failure; last-admin demotion is refused in the service layer. FE: role control added to member rows (disabled for the current user and for non-managers), aligned `MemberRole`/`getRoleConfig`/`DEFAULT_ROLES` to `admin|approver|member`, fixing the existing approver mislabel.

- **Rationale:** role must take effect at the authorization boundary (Stytch), not just the local copy.

### 5. Header buttons — wire or remove
Preferences → `/dashboard/settings`; Support → configured contact (mailto) if present, else remove the button. No dangling inert controls.

### 6. Audit log view — FE-only layer over existing activities endpoint
Add `audit` to the `SettingsView` union, `parseViewParam`, `DETAIL_META`, and `renderDetailContent` branch in `settings-content.tsx`, gated by `hasPermission(PERMISSIONS.AUDIT_VIEW)`. The view renders a new `AuditLogView` component that reuses the existing `useActivitiesQuery({ tipo, limit, offset })` hook and `ActivityDto` model — no backend, DB, or migration changes.

```
settings?view=audit
  └─ AuditLogView (gated audit:view)
       └─ useActivitiesQuery({tipo, limit, offset})
            └─ GET /api/crm/actividades   (backend enforces contact:view)
```

- **Data source:** `GET /api/crm/actividades` already returns unified org activity (`ActivityWithActor`: tipo, asunto, contenido, performed-by name, timestamps, entity refs) with `tipo` filter and `limit`/`offset` pagination — verified in `internal/modules/crm/routes.go` (lines 65-71) and the `activity-timeline` spec.
- **Gate split:** frontend hides the view without `audit:view`; the backend route continues enforcing `contact:view`. `audit:view` is cosmetic today (no backend route consumes it) — accepted for MVP, noted in proposal.
- **Alternative rejected:** building a new `GET /api/agent/actions` audit endpoint and merging agent rows — larger backend scope, contradicts "no backend changes" non-goal.
- **Alternative rejected:** a dedicated sidebar page `/dashboard/audit` — adds nav + routing surface; the settings view-stack pattern already exists and matches the other admin features in this change.

### 7. Stytch circuit breaker on the outbound sync path
The new org/account sync calls are the first org-scoped `PUT` flows that mutate Stytch, so the governance rule (fallback/circuit-breaker state for every outbound Stytch SDK invocation) applies. The shared Stytch client (`internal/platform/stytch/client.go`) previously issued SDK calls without any breaker. Add a two-tier circuit breaker (threshold 5, cooldown 10s, half-open probe 2) to the client and guard every outbound call with `Run(ctx, fn)`: when open, the call fails fast with a breaker error, and the calling service rejects the request without any local write (no SSOT drift). Breaker parameters are config-driven with the above defaults.

- **Rationale:** without the breaker, a degraded Stytch endpoint would take full API timeouts on every admin save instead of failing fast; governance requires explicit breaker state.
- **Alternative rejected:** leaving Stytch calls unguarded — violates the constitution's governance rules and the AGENTS.md invariants.

## Risks / Trade-offs

- [Dual-SSOT drift if Stytch sync fails] → Reject the request on Stytch failure; never commit local-only writes; circuit breaker already guards the Stytch call path.
- [Role vocabulary migration (`manager` → `approver`) breaks existing FE role configs] → Update `MemberRole`, `getRoleConfig`, `DEFAULT_ROLES`, and any `manager` references in one commit; backend already treats `approver` as the canonical local role.
- [Last-admin demotion locks org out of management] → Enforce at the service layer (refuse demotion when the member is the only remaining admin); FE also disables the option and shows an inline error.
- [WhatsApp embedded signup untested at runtime (never mounted)] → Add e2e coverage exercising the no-config → connect → connected path with the existing mock/backend fixtures.

## Migration Plan

1. Deploy backend first (extended `UpdateOrganization`/`UpdateAccount` semantics are backward-compatible with existing callers; no migration of stored data).
2. Deploy frontend nav + settings wiring + header changes.
3. Deploy member-role vocabulary alignment (single commit touching model + UI).
4. Rollback: git revert per commit; re-issue prior Stytch org display name / member roles via the same extended endpoints for any mutated rows.

## Open Questions

- ~~Confirm the exact Stytch B2B method/fields available in the wrapped SDK adapter for `Organizations.Update` display_name and `Members.Update` role (stytch-go v18).~~ — RESOLVED: `Organizations.Update` (`stytch_organization_repository.go:122`) and `Members.Update` via `AssignRoles` (`stytch_member_repository.go:191`); circuit breaker added in Decision 7.
- ~~Confirm whether any org besides the current one writes to `PUT /organizations` before extending it.~~ — RESOLVED: only `PUT /organizations` for the current org (`routes.go:75`) writes this row; no other callers found.
