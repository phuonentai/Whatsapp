## MODIFIED Requirements

### Requirement: Login page renders a custom email form
The `/auth` login page SHALL render the pre-built Stytch B2B component (`<StytchB2B />` from `@stytch/nextjs/b2b`) in Discovery mode instead of a custom email form, configured with `locale: "es"` and any required strings overrides. The component SHALL render the project-enabled primary methods (email magic link, OAuth social, email OTP, passkeys) and the org discovery surface. The custom email-only form with membership pre-validation (`members.search` check + `sendMagicLink` pre-check) SHALL NOT be rendered on `/auth`. Unknown emails reach an organization only via invite or the organization's opted-in domain-restricted JIT join (see `stytch-login-surface`).

#### Scenario: Known member submits email

- **WHEN** a browser submits an email that belongs to an existing member
- **THEN** the Stytch component SHALL handle the magic-link send via the discovery flow
- **AND** the member SHALL complete authentication at `/authenticate`

#### Scenario: Unknown member submits email

- **WHEN** a browser submits an email with no matching organization (no membership, no JIT-enabled organization)
- **THEN** the discovery flow SHALL surface no joinable organization for that address
- **AND** the discovery result SHALL NOT reveal any organization to the unknown address (the send itself is governed by Stytch's per-email rate limiting; the platform's in-process limiter does not apply to client-side discovery sends)

#### Scenario: No password input

- **WHEN** a browser visits `/auth`
- **THEN** the page SHALL NOT contain any custom form components (the Stytch component owns the form) or `input[type="password"]` elements

### Requirement: Settings uses custom member management
The `/settings` page SHALL render the application's custom member management components (`member-list.tsx` and `invite-member.tsx`) for member and role management, and SHALL NOT depend on Stytch AdminPortal components for member management. SSO and SCIM management views (`?view=sso`, `?view=scim`) SHALL render the Stytch Admin Portal SSO and SCIM components per the `enterprise-sso` and `scim-provisioning` capabilities (subject to the Admin Portal availability gate in `enterprise-sso`/design D4).

#### Scenario: Admin invites a member

- **WHEN** an admin submits the invite form for a new member
- **THEN** the invite SHALL be processed through the application's invite flow (Stytch `SendInvite`)
- **AND** the member SHALL appear in the custom member list after acceptance

#### Scenario: Settings renders Admin Portal for SSO/SCIM only

- **WHEN** a member with `org:manage` opens `/settings?view=sso` or `/settings?view=scim`
- **THEN** the page SHALL render the corresponding Admin Portal component
- **AND** member management SHALL remain the custom `member-list.tsx` / `invite-member.tsx` components
