# Reconcile Auth Spec Drift

## Why

The living spec `stytch-nextjs-components` describes behavior that the implementation deliberately does not have, and the implementation has deliberate behavior the spec does not describe:

- Spec: `/login` renders `<StytchB2B />` (Discovery flow, email magic links + SSO) with zero custom form elements. Code: there is no `/login` route; `/auth` renders a custom email form (`app/auth/page.tsx`) that pre-validates membership via Stytch `members.search` before sending a magic link.
- Spec: `/settings` renders `<AdminPortalMemberManagement />` and `<AdminPortalSSO />`. Code: `/settings` uses custom member management (`member-list.tsx`, `invite-member.tsx`); no AdminPortal components, no SSO UI.
- The custom flow is a documented, deliberate security decision (`STYTCH_CONFIGURATION.md`): prevent magic-link emails to unknown users, block JIT provisioning, restrict discovery — the pre-built component always sends emails and would leak/burn sends.
- Additionally, `STYTCH_CONFIGURATION.md` references routes that no longer exist (`/api/auth/magic-link`, `/api/auth/logout` migrated to server actions), and its session-duration value (43200) contradicts the code default (480) — the latter is fixed by the `session-lifetime-hardening` change; this change fixes the stale route references.

Per AGENTS.md, where code and spec disagree, the spec wins OR a change proposal reconciles the spec. This change reconciles: the spec is amended to describe the actual deliberate design, with the SSO intent preserved as a future requirement.

## What Changes

- **MODIFIED `stytch-nextjs-components`:** replace the StytchB2B-component login requirement with the custom email-form design (membership pre-validation, neutral failure, no password, SSO product enabled for future surfacing); replace the AdminPortal settings requirement with the custom member-management design.
- **REMOVED `stytch-nextjs-components`:** the "no custom form components" and AdminPortal-specific requirements that contradict the deliberate design.
- **Docs:** fix stale route references in `STYTCH_CONFIGURATION.md` (server actions replace `/api/auth/magic-link` and `/api/auth/logout`).

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `stytch-nextjs-components`: requirements updated to match the deliberate custom-flow design.

## Impact

- **Governance only:** no production code changes. Spec text + `STYTCH_CONFIGURATION.md` route references.
- **Frontend:** none (no behavior change).
- **Backend:** none.
- **Stytch:** none (no tenant policy changes).

## Rollback

- **Git:** revert the spec/docs edits.
- **Stytch tenant policy state:** none involved.

## Non-Goals

- NOT implementing the pre-built `<StytchB2B />` or AdminPortal components (explicitly rejected — the custom flow is the security baseline).
- NOT changing auth behavior; purely reconciling governance artifacts with reality.
- NOT altering the `signup-stytch-compliance`, `edge-middleware-session`, or `auth-passwordless-e2e` specs (they already match implementation).
