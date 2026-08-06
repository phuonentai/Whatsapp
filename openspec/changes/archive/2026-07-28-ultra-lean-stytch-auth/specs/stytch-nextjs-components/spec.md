## ADDED Requirements

### Requirement: Login page renders pre-built Stytch B2B component

The system SHALL render `<StytchB2B />` from `@stytch/nextjs/b2b` on the `/login` page with Discovery auth flow, Email Magic Links, and SSO products enabled.

#### Scenario: Login page loads Stytch component

- **WHEN** a user navigates to `/login`
- **THEN** the page SHALL render the `<StytchB2B />` component
- **AND** the auth flow type SHALL be `AuthFlowType.Discovery`
- **AND** `B2BProducts.emailMagicLinks` and `B2BProducts.sso` SHALL be enabled

#### Scenario: User completes magic link login

- **WHEN** a user submits their email on the login page
- **AND** Stytch sends a magic link email
- **AND** the user clicks the magic link
- **THEN** Stytch redirects the user to the authenticated session
- **AND** the `stytch_session_jwt` cookie SHALL be set

### Requirement: Settings page renders Stytch admin portal components

The system SHALL render `<AdminPortalMemberManagement />` and `<AdminPortalSSO />` from `@stytch/nextjs/b2b` on the `/settings` page.

#### Scenario: Settings page loads admin components

- **WHEN** an authenticated admin user navigates to `/settings`
- **THEN** the page SHALL render the `<AdminPortalMemberManagement />` component for member invites and role management
- **AND** the page SHALL render the `<AdminPortalSSO />` component for SSO configuration

#### Scenario: Admin invites a member

- **WHEN** an admin user fills the invite form in `<AdminPortalMemberManagement />` and submits
- **THEN** Stytch SHALL send an invite magic link email to the new member
- **AND** the member SHALL appear in the member list

### Requirement: No custom form components in auth pages

The `/login` and `/settings` pages SHALL NOT contain custom form components, password fields, or session generation code.

#### Scenario: Login page has zero custom form elements

- **WHEN** inspecting the `/login` page source
- **THEN** there SHALL be no `<form>`, `<input>`, or custom button elements outside the `<StytchB2B />` component
- **AND** no password-related fields or validation logic SHALL exist in the page component
