## MODIFIED Requirements

### Requirement: Password authentication resolves organization before Stytch call

The system SHALL authenticate members using email and password via Stytch's `Passwords.Authenticate` endpoint.

Before calling the Stytch API, the system MUST resolve the member's Stytch `organization_id` by:
1. Looking up the member's account by email in the local PostgreSQL database
2. Retrieving the associated Stytch organization ID from the local org mapping
3. Passing the resolved organization ID in the `Passwords.Authenticate` request

An empty `OrganizationID` MUST NOT be passed to the Stytch API.

#### Scenario: Successful password login with resolved org
- **WHEN** a member sends `POST /auth/login` with valid email and password
- **THEN** the system looks up the member's account by email in the local DB
- **AND** resolves the Stytch organization ID from the org mapping
- **AND** calls Stytch `Passwords.Authenticate` with the resolved `organization_id`
- **AND** returns a session token and session JWT

#### Scenario: Email not found locally
- **WHEN** a member sends `POST /auth/login` with an email not found in the local database
- **THEN** the system returns a 401 Unauthorized error with code `INVALID_CREDENTIALS`

#### Scenario: Stytch org ID not mapped
- **WHEN** the member's account exists locally but has no Stytch org mapping
- **THEN** the system returns a 401 Unauthorized error with code `INVALID_CREDENTIALS`