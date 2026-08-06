## ADDED Requirements

### Requirement: Auth page uses same-origin API calls

The auth page login flow SHALL use a same-origin relative URL (`/api`) as the default base URL for backend API calls, consistent with the `ApiClient` pattern.

#### Scenario: Email check with unset API base URL env var

- **WHEN** `NEXT_PUBLIC_API_BASE_URL` is not set
- **THEN** the email check fetch SHALL use `/api/auth/check-email` (relative, same-origin) instead of `http://localhost:8080/api/auth/check-email` (cross-origin)

#### Scenario: Email check with custom API base URL env var

- **WHEN** `NEXT_PUBLIC_API_BASE_URL` is set to a custom absolute URL
- **THEN** the email check fetch SHALL use the configured URL as before (no behavioral change)
