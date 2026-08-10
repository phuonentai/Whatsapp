## Purpose

Defines Stytch-compliant signup: native invite organization bootstrap, no owner_password field, and structured error codes.

## Requirements

### Requirement: Organization bootstrap uses Stytch native invite flow

The system SHALL create the owner member in Stytch with `SendInvite: true` on the `Members.Create` API call, allowing Stytch to handle the invite email natively.

The system MUST NOT call `MagicLinks.Email.LoginOrSignup` separately after member creation during organization bootstrap.

If the `CreateMember` call with `SendInvite: true` succeeds, the `MagicLinkSent` field in the bootstrap response SHALL be `true`.

If the `CreateMember` call fails before the invite is sent, the `MagicLinkSent` field SHALL be `false` and the error MUST be returned to the caller.


#### Scenario: Successful bootstrap with Stytch native invite

- **WHEN** a valid signup request is received
- **AND** the org creation and local DB setup succeed
- **AND** `Members.Create` with `SendInvite: true` succeeds
- **THEN** the response `MagicLinkSent` is `true`
- **AND** no separate `LoginOrSignup` API call is made
- **AND** Stytch sends the invite magic link email to the owner

#### Scenario: Member creation fails during bootstrap

- **WHEN** a valid signup request is received
- **AND** `Members.Create` with `SendInvite: true` returns an error
- **THEN** the bootstrap operation fails
- **AND** the response `MagicLinkSent` is `false`
- **AND** the error is returned to the handler with a descriptive message
- **AND** previously created resources (org, local org) are rolled back

#### Scenario: Invite send rejects due to invalid Stytch config

- **WHEN** `Members.Create` is called with `SendInvite: true`
- **AND** the invite redirect URL is not configured in the Stytch dashboard
- **THEN** Stytch returns an error
- **AND** the system returns a 500 with code `INVITE_FAILED` and detail referencing the missing redirect URL configuration

### Requirement: No `owner_password` field in signup payload

The frontend signup request SHALL NOT include an `owner_password` field.

The `SignupMagicLinkRequestDto` in the frontend SHALL NOT define an `owner_password` property.

The signup repository SHALL NOT generate or transmit a password as part of the organization bootstrap request.

#### Scenario: Signup request excludes password

- **WHEN** the frontend submits a signup request
- **THEN** the payload does not contain an `owner_password` field
- **AND** the Go backend successfully binds the request without the field

#### Scenario: Password generator is no longer used by signup

- **WHEN** the `password-generator.ts` utility file exists in the codebase
- **THEN** it SHALL NOT be imported or called by the signup repository or any signup-related code

### Requirement: Structured error codes in signup responses

The system SHALL return signup errors with a machine-readable `code` field and a human-readable `detail` field in the JSON response body.

The following error codes MUST be recognized:

| Code | Condition |
|------|-----------|
| `STYTCH_UNAUTHORIZED` | Stytch API returns 401 (invalid credentials) |
| `STYTCH_UNREACHABLE` | Stytch API is unreachable (timeout, connection refused) |
| `SLUG_CONFLICT` | Organization slug generation exhausted all retries |
| `DB_CONNECTION_FAILED` | PostgreSQL is unreachable |
| `INVITE_FAILED` | Stytch member invite (SendInvite: true) returned an error |
| `INVALID_REQUEST` | Request validation failed (missing fields, bad email) |

#### Scenario: Stytch credential error produces structured error

- **WHEN** a signup request triggers a Stytch API call
- **AND** Stytch returns 401 Unauthorized
- **THEN** the response has HTTP status 500
- **AND** the response body contains `"code": "STYTCH_UNAUTHORIZED"`
- **AND** the response body contains a `detail` field suggesting credential verification

#### Scenario: Valid request with valid credentials succeeds without error code

- **WHEN** a signup request succeeds
- **THEN** the response has HTTP status 201
- **AND** the response body does not contain an `error` or `code` field
