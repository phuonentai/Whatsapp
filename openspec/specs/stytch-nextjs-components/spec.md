## Purpose

Define the custom authentication frontend: `/auth` renders a custom email-only form with membership pre-validation and anti-enumeration, and `/settings` uses the application's custom member management components. Pre-built Stytch B2B / AdminPortal component mandates are removed.

## Requirements

### Requirement: Login page renders a custom email form
The `/auth` login page SHALL render a custom email-only form (no password input) that validates membership via the Stytch B2B `members.search` API before sending a magic link. For emails with no matching member, the page SHALL show the identical neutral message as for existing members and SHALL NOT call `magicLinks.email.loginOrSignup`.

#### Scenario: Known member submits email

- **WHEN** a browser submits an email that belongs to an existing member
- **THEN** the system SHALL send a magic link via `magicLinks.email.loginOrSignup` for the member's organization(s)
- **AND** SHALL display the neutral "If an account exists with that email, a magic link has been sent." message

#### Scenario: Unknown member submits email

- **WHEN** a browser submits an email with no matching member
- **THEN** the system SHALL NOT call the magic-link send API
- **AND** SHALL display the same neutral message (no enumeration)

#### Scenario: No password input

- **WHEN** a browser visits `/auth`
- **THEN** the page SHALL NOT contain any `input[type="password"]` element

### Requirement: Settings uses custom member management
The `/settings` page SHALL render the application's custom member management components (`member-list.tsx` and `invite-member.tsx`) and SHALL NOT depend on Stytch AdminPortal UI components.

#### Scenario: Admin invites a member

- **WHEN** an admin submits the invite form for a new member
- **THEN** the invite SHALL be processed through the application's invite flow (Stytch `SendInvite`)
- **AND** the member SHALL appear in the custom member list after acceptance
