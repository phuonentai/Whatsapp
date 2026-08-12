## MODIFIED Requirements

### Requirement: RBAC API endpoint authentication

All RBAC API endpoints (roles, permissions, by-category, role details, check-permission, metadata) SHALL require a valid authenticated session. Unauthenticated requests MUST receive a 401 Unauthorized response.

The frontend RBAC repository SHALL authenticate every RBAC API request with the user's session. `skipAuth` MUST NOT be used for RBAC endpoints.

#### Scenario: Authenticated RBAC request

- **WHEN** an authenticated user sends a GET request to `/api/rbac/roles`
- **THEN** the system returns a 200 response with the roles and their permissions

#### Scenario: Unauthenticated RBAC roles request

- **WHEN** an unauthenticated user sends a GET request to `/api/rbac/roles`
- **THEN** the system returns a 401 Unauthorized response

#### Scenario: Unauthenticated RBAC check-permission request

- **WHEN** an unauthenticated user sends a POST request to `/api/rbac/check-permission`
- **THEN** the system returns a 401 Unauthorized response

#### Scenario: Unauthenticated RBAC metadata request

- **WHEN** an unauthenticated user sends a GET request to `/api/rbac/metadata`
- **THEN** the system returns a 401 Unauthorized response

#### Scenario: Frontend RBAC fetch attaches the session

- **WHEN** the frontend fetches RBAC roles via the RBAC repository
- **THEN** the request carries the user's session access token
- **AND** an unauthenticated response is handled by the standard 401 session-expiry flow (redirect to login), never silently ignored

#### Scenario: Frontend RBAC fetch without session

- **WHEN** the frontend has no valid session and fetches RBAC roles
- **THEN** the backend returns 401
- **AND** the frontend redirects the user to the login page
