## ADDED Requirements

### Requirement: Email existence check endpoint

The system SHALL expose `GET /api/auth/check-email?email=<address>` that returns whether an active account exists for the given email, resolved by a read-only local database lookup (organizations table via the organization repository) without calling Stytch B2B APIs and without requiring an authenticated session. The endpoint SHALL NOT return or store any credential, password, MFA, or session material.

#### Scenario: Email exists

- **WHEN** a client sends `GET /api/auth/check-email?email=admin@example.com` and an active account with that email exists in the local organizations table
- **THEN** the system SHALL return HTTP 200 with an empty response body

#### Scenario: Email not found

- **WHEN** a client sends `GET /api/auth/check-email?email=nobody@example.com` and no active account with that email exists
- **THEN** the system SHALL return HTTP 404 with an error body

#### Scenario: Missing email parameter

- **WHEN** a client sends `GET /api/auth/check-email` without an `email` query parameter
- **THEN** the system SHALL return HTTP 400 with an error body

#### Scenario: Repository failure

- **WHEN** the local repository lookup fails for a reason other than "not found"
- **THEN** the system SHALL return HTTP 500 with an error body

#### Scenario: No session required

- **WHEN** a client sends `GET /api/auth/check-email?email=admin@example.com` without any session cookies or auth headers
- **THEN** the system SHALL perform the lookup and return HTTP 200 (or 404) without requiring authentication

### Requirement: Auth page resolves API base URL through the shared client configuration

The auth page login flow SHALL resolve the backend API base URL through the shared `ApiClient` configuration (`apiClient.getBaseUrl()`), which defaults to a same-origin relative `/api` on the client, and SHALL NOT hardcode a cross-origin fallback such as `http://localhost:8080/api` for the email existence check.

#### Scenario: Email check uses same-origin default

- **WHEN** `NEXT_PUBLIC_API_BASE_URL` is not set and the auth page performs the email existence check in a browser
- **THEN** the check SHALL be sent to a same-origin relative URL (`/api/auth/check-email`) instead of a hardcoded cross-origin URL

#### Scenario: Custom API base URL respected

- **WHEN** `NEXT_PUBLIC_API_BASE_URL` is set to a custom absolute URL
- **THEN** the email check SHALL use the configured URL as before (no behavioral change)
