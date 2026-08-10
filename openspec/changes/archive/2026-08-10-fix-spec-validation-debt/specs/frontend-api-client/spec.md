## ADDED Requirements

### Requirement: ApiClient singleton with typed HTTP verbs

The frontend SHALL use a single `ApiClient` instance (`apiClient`) exposing typed `get`, `post`, `put`, `patch`, and `delete` methods over the global `fetch`. The client SHALL resolve its base URL at construction: server-side from `API_BASE_URL_INTERNAL` (default `http://localhost:8080/api`), client-side from `NEXT_PUBLIC_API_BASE_URL` (default relative `/api`). All requests SHALL be sent with `credentials: "include"`.

#### Scenario: Server-side request uses internal base URL

- **WHEN** `apiClient.get("/modules")` executes in a server context
- **AND** `API_BASE_URL_INTERNAL` is set
- **THEN** the request SHALL target `API_BASE_URL_INTERNAL + /modules`

#### Scenario: Client-side request uses relative base URL

- **WHEN** `apiClient.get("/modules")` executes in a browser context
- **AND** `NEXT_PUBLIC_API_BASE_URL` is unset
- **THEN** the request SHALL target `/api/modules` relative to the origin

#### Scenario: Request body JSON serialization

- **WHEN** a `post`/`put`/`patch` call receives a plain object body
- **THEN** the body SHALL be JSON-serialized
- **AND** `Content-Type: application/json` SHALL be set unless an explicit header overrides it
- **AND** `FormData` bodies SHALL be passed through unmodified

### Requirement: Bearer token attachment and session refresh

The client SHALL attach the Stytch access token as `Authorization: Bearer <token>` unless the caller passes `skipAuth: true` or an explicit `Authorization` header. On a 401 response, the client SHALL refresh the token (in-memory cache first, then cookie, then Stytch exchange server-side or `/api/auth/session/refresh` browser-side), retry the request once, and — if refresh fails — clear session cookies and redirect to the login route with a `returnTo` query parameter.

#### Scenario: Token resolved from memory cache

- **WHEN** a request is made without `skipAuth`
- **AND** a valid, non-expired token is cached in memory
- **THEN** the request SHALL include `Authorization: Bearer <cached-token>`

#### Scenario: Token resolved from stored cookie

- **WHEN** a request is made without `skipAuth`
- **AND** no valid in-memory cache exists
- **AND** `SESSION_JWT_COOKIE_NAME` holds a non-expired JWT
- **THEN** the request SHALL use that JWT
- **AND** the JWT SHALL be persisted to the browser session cookie

#### Scenario: Token refresh with retry and backoff

- **WHEN** the client must refresh the access token
- **THEN** the refresh SHALL retry up to 3 times with exponential backoff (1s, 2s, 4s)
- **AND** concurrent callers SHALL share a single refresh promise bounded by a 10-second timeout

#### Scenario: 401 with successful refresh

- **WHEN** a request returns 401
- **AND** the client obtains a fresh token
- **THEN** the original request SHALL be retried once with the new token

#### Scenario: 401 with failed refresh

- **WHEN** a request returns 401
- **AND** token refresh fails
- **THEN** session cookies SHALL be cleared
- **AND** the client SHALL redirect to the login route with the current path as `returnTo`

#### Scenario: Server-side token exchange

- **WHEN** token refresh executes server-side
- **AND** a Stytch session token cookie exists
- **THEN** the client SHALL exchange it for a session JWT via the Stytch B2B `sessions.authenticate` API with an 8-hour session duration

### Requirement: Mock-auth E2E identity forwarding

When `AUTH_MOCK_ENABLED === "true"`, the client SHALL forward the `X-Test-Org-ID` identity header on every request that does not already carry one, sourced from the `X-Test-Org-ID` cookie (server: `next/headers` cookies; browser: `document.cookie`). Missing or unreadable cookies SHALL be ignored without failing the request.

#### Scenario: Server-side mock identity read

- **WHEN** `AUTH_MOCK_ENABLED` is true
- **AND** a server-side request lacks `X-Test-Org-ID`
- **THEN** the client SHALL read the cookie from the request cookies
- **AND** forward it as `X-Test-Org-ID` on the outgoing request

#### Scenario: Client-side mock identity read

- **WHEN** `AUTH_MOCK_ENABLED` is true
- **AND** a browser request lacks `X-Test-Org-ID`
- **THEN** the client SHALL read the cookie from `document.cookie`
- **AND** forward it as `X-Test-Org-ID` on the outgoing request

### Requirement: JSON envelope unwrapping in repositories

Domain repositories SHALL unwrap the backend `{ data?: T; success?: boolean }` envelope returned by list and entity endpoints, returning the inner `data` payload. When a response is not an envelope (no `data` property), the repository SHALL return the response as-is. Repository endpoints SHALL be declared as typed constants (e.g., `const BASE = "/path"`) with methods returning typed DTOs.

#### Scenario: Enveloped list response unwrapped

- **WHEN** a repository method receives `{ data: [...], success: true }`
- **THEN** the method SHALL return the inner array

#### Scenario: Non-envelope response passed through

- **WHEN** a repository method receives a response without a `data` property
- **THEN** the method SHALL return the response unmodified

#### Scenario: Error responses become typed errors

- **WHEN** a request returns a non-OK status
- **THEN** the client SHALL throw an `API Error <status>: <message>` error
- **AND** the message SHALL be taken from the response body's `message` field when present, else the HTTP status text
