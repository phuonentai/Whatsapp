## REMOVED Requirements

### Requirement: CRM page is accessible from sidebar navigation

**Reason**: The sidebar navigation is not affected by this change. Auth is handled at the middleware layer, not per-page.

**Migration**: No action needed. Sidebar visibility continues to be gated by subscription and permissions as before.

### Requirement: CRM dashboard is an SPA with view navigation in Spanish

**Reason**: CRM SPA behavior is unrelated to auth/session changes.

**Migration**: No action needed. CRM SPA behavior continues unchanged.

## ADDED Requirements

### Requirement: Auth pages use pre-built Stytch components

The system SHALL replace any custom auth-related pages (`/login`, `/signup`, `/settings/members`, `/settings/sso`) with Stytch pre-built B2B components from `@stytch/nextjs/b2b`.

#### Scenario: Login page is the Stytch component

- **WHEN** a user navigates to `/login`
- **THEN** the page SHALL render `<StytchB2B />` with Discovery flow
- **AND** NO custom form components SHALL exist on the page

#### Scenario: Settings page renders Stytch admin portal

- **WHEN** an authenticated admin navigates to `/settings`
- **THEN** the page SHALL render `<AdminPortalMemberManagement />` and `<AdminPortalSSO />`
- **AND** NO custom member management or SSO form components SHALL exist on the page

### Requirement: Protected route gating via edge middleware

The system SHALL gate all protected frontend routes (`/dashboard/:path*`, `/settings/:path*`) through the edge middleware instead of client-side or server-side per-page checks.

#### Scenario: Unauthenticated user redirected from dashboard

- **WHEN** an unauthenticated user navigates to `/dashboard/crm`
- **THEN** the edge middleware SHALL redirect to `/login`
- **AND** no CRM page code SHALL execute
