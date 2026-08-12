## MODIFIED Requirements

### Requirement: Auth pages use pre-built Stytch components
The system SHALL use Stytch pre-built B2B components for the login surface and the SSO/SCIM management surfaces: `/auth` SHALL render `<StytchB2B />` from `@stytch/nextjs/b2b` with the Discovery flow (organization creation hidden), and `/settings` SHALL render `<AdminPortalSSO />` and `<AdminPortalSCIM />` for SSO and SCIM management. Member and role management SHALL remain the application's custom components (`member-list.tsx`, `invite-member.tsx`, equipo y permisos) and SHALL NOT be replaced by `AdminPortalMemberManagement`.

#### Scenario: Login page is the Stytch component

- **WHEN** a user navigates to `/auth`
- **THEN** the page SHALL render `<StytchB2B />` with Discovery flow
- **AND** NO custom form components SHALL exist on the page

#### Scenario: Settings page renders Stytch admin portal for SSO/SCIM

- **WHEN** an authenticated admin navigates to `/settings?view=sso` or `/settings?view=scim`
- **THEN** the page SHALL render `<AdminPortalSSO />` or `<AdminPortalSCIM />` respectively
- **AND** NO custom SSO or SCIM form components SHALL exist on those views

#### Scenario: Member management remains custom

- **WHEN** an admin manages members or roles in `/settings`
- **THEN** the application's custom member management components SHALL be used
- **AND** `AdminPortalMemberManagement` SHALL NOT be rendered
