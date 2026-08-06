## ADDED Requirements

### Requirement: Signup with password

The system SHALL allow a user to create an organization with an owner member, setting a password during registration instead of sending an invite email.

#### Scenario: Successful signup with password

- **WHEN** a user submits the signup form with `org_display_name`, `owner_email`, `owner_name`, and `owner_password`
- **THEN** the system SHALL create the organization in Stytch, create the local organization record, create the Stytch member without sending an invite, set the member's password via Stytch's password API, assign the admin role, and create the local account record
- **THEN** the response SHALL include `success: true`, `organization_id`, `owner_member_id`, and `owner_email`

#### Scenario: Signup with weak password

- **WHEN** a user submits a password shorter than 8 characters
- **THEN** the system SHALL return a validation error with `code: "PASSWORD_TOO_WEAK"`

#### Scenario: Signup with duplicate email in Stytch

- **WHEN** a user submits a signup with an email that already has a Stytch member
- **THEN** the system SHALL roll back the organization creation and return an error with `code: "DUPLICATE_EMAIL"`

### Requirement: Login with email and password

The system SHALL authenticate a member using their email and password via Stytch `Passwords.Authenticate`.

#### Scenario: Successful login

- **WHEN** a user submits `POST /api/auth/login` with valid `email` and `password`
- **THEN** the system SHALL verify the credentials via Stytch `Passwords.Authenticate`
- **THEN** the response SHALL include `session_token`, `session_jwt`, and member details

#### Scenario: Login with incorrect password

- **WHEN** a user submits `POST /api/auth/login` with an incorrect password
- **THEN** the system SHALL return an error with `code: "INVALID_CREDENTIALS"`

### Requirement: Magic link fallback

The system SHALL retain the ability to send a magic link for users who prefer it over password login.

#### Scenario: Send magic link on login page

- **WHEN** a user clicks "Send magic link" on the login page
- **THEN** the existing `sendMagicLink` flow SHALL still work
