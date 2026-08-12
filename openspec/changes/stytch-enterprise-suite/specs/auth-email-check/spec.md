## MODIFIED Requirements

### Requirement: Auth page resolves API base URL through the shared client configuration

Any client of the email existence check SHALL resolve the backend API base URL through the shared `ApiClient` configuration (`apiClient.getBaseUrl()`), which defaults to a same-origin relative `/api` on the client, and SHALL NOT hardcode a cross-origin fallback such as `http://localhost:8080/api`. The `/auth` login page SHALL NOT perform the email existence check: the pre-built Stytch B2B component owns the login form (see `stytch-login-surface`), and the endpoint's clients are any remaining callers (for example tests) found by the task-4.2 caller audit.

#### Scenario: Email check uses same-origin default

- **WHEN** `NEXT_PUBLIC_API_BASE_URL` is not set and a remaining client of the email existence check performs the check in a browser
- **THEN** the check SHALL be sent to a same-origin relative URL (`/api/auth/check-email`) instead of a hardcoded cross-origin URL

#### Scenario: Custom API base URL respected

- **WHEN** `NEXT_PUBLIC_API_BASE_URL` is set to a custom absolute URL
- **THEN** the email check SHALL use the configured URL as before (no behavioral change)

#### Scenario: Auth page no longer performs the check

- **WHEN** a browser visits `/auth`
- **THEN** the page SHALL render the pre-built Stytch B2B component in Discovery mode
- **AND** the page SHALL NOT issue `GET /api/auth/check-email` or any custom membership pre-validation call
