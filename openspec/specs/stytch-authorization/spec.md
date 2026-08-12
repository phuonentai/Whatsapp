## Purpose

Define how Stytch RBAC policies drive authorization in the Go backend: permission resolution, role normalization, RBAC API authentication, and the edge-middleware trust boundary.
## Requirements
### Requirement: stytch-go upgraded from v16 to v18

The system SHALL use `github.com/stytchauth/stytch-go/v18` as the Stytch Go SDK dependency.

All Stytch API client calls in the Go backend SHALL be compatible with the v18 SDK package structure and method signatures.

The `LoginOrSignup` magic link method SHALL NOT be called during organization bootstrap. Member invite emails SHALL be sent via `Members.Create` with `SendInvite: true`.

#### Scenario: stytch-go v18 compiles and all existing Stytch API calls work

- **WHEN** the Go module is updated to use stytch-go v18
- **AND** `go build` is executed
- **THEN** compilation succeeds without import or signature errors
- **AND** organization creation, member management, RBAC, and session operations remain functional

### Requirement: Permission resolution from Stytch RBAC policy

The system SHALL resolve all role-to-permission mappings exclusively from the Stytch RBAC policy API. Hardcoded role→permission maps in Go code MUST NOT be used as a source of truth for authorization decisions.

The Stytch RBAC policy MUST be cached in Redis with a 5-minute TTL. On cache hit, the cached policy MUST be used without calling the Stytch API. On cache miss, the policy MUST be fetched from Stytch, cached, and then used.

If the Stytch RBAC API is unreachable and the cache is empty, the system MUST return a 503 Service Unavailable error for any authorization check.

#### Scenario: Permission check with cached policy
- **WHEN** a request requires permission verification
- **AND** the Stytch RBAC policy is in the Redis cache
- **THEN** the system reads permissions from the cached policy
- **AND** the system does NOT call the Stytch API

#### Scenario: Permission check with cache miss
- **WHEN** a request requires permission verification
- **AND** the Stytch RBAC policy is NOT in the Redis cache
- **THEN** the system fetches the policy from the Stytch RBAC API
- **AND** caches it in Redis with a 5-minute TTL
- **AND** resolves the permission from the fetched policy

#### Scenario: Permission check with Stytch API unavailable
- **WHEN** a request requires permission verification
- **AND** the Stytch RBAC policy is NOT in the Redis cache
- **AND** the Stytch RBAC API is unreachable
- **THEN** the system returns a 503 Service Unavailable error

### Requirement: Go backend accepts pre-validated JWT from edge middleware

The Go backend SHALL accept the `X-Forwarded-Auth: true` header set by the Next.js edge middleware. When this header is present and the `X-Stytch-Organization-Id` and `X-Stytch-Member-Id` headers carry valid UUIDs, the Go session middleware MAY skip redundant Stytch API token introspection for performance.

The system SHALL still validate the JWT signature independently if `X-Forwarded-Auth` is absent or the headers are malformed. The Go backend MUST NOT trust unvalidated headers.

#### Scenario: Request arrives with valid X-Forwarded-Auth headers

- **WHEN** a request arrives at the Go API
- **AND** `X-Forwarded-Auth: true` is set
- **AND** `X-Stytch-Organization-Id` and `X-Stytch-Member-Id` are valid UUIDs
- **THEN** the Go session middleware SHALL use these headers as the authenticated context
- **AND** MAY skip calling Stytch API for token introspection
- **AND** the `organization_id` and `member_id` SHALL be set on the request context

#### Scenario: Request arrives without X-Forwarded-Auth

- **WHEN** a request arrives at the Go API
- **AND** `X-Forwarded-Auth` is absent or `false`
- **THEN** the Go session middleware SHALL validate the JWT or session token independently via the Stytch API (existing behavior)
- **AND** MUST NOT trust any user-supplied `organization_id` or `member_id` headers

#### Scenario: X-Forwarded-Auth is present but headers are malformed

- **WHEN** a request arrives at the Go API
- **AND** `X-Forwarded-Auth: true` is set
- **AND** `X-Stytch-Organization-Id` is not a valid UUID
- **THEN** the Go session middleware SHALL reject the request with a 401 Unauthorized response
- **AND** SHALL log a warning about malformed edge-auth headers

### Requirement: Log mismatches between edge and Go validation

The Go backend SHALL log a warning when the edge middleware's `X-Forwarded-Auth` and the Go backend's independent Stytch API validation produce different results (e.g., JWT valid at edge but rejected by Stytch API).

#### Scenario: Edge says valid, Go says invalid

- **WHEN** a request arrives with `X-Forwarded-Auth: true`
- **AND** the Go backend attempts independent validation via Stytch API
- **AND** Stytch API returns an invalid/expired session response
- **THEN** the Go backend SHALL log a warning with the `stytch_member_id`, `stytch_organization_id`, and Stytch API error
- **AND** SHALL return a 401 Unauthorized

### Requirement: RBAC export action for bulk download

The Stytch RBAC policy SHALL define an `export` action on the `contact`, `deal`, and `activity` resources, and the `admin` and `manager` roles SHALL be granted it, so that bulk data download is a distinct, explicitly-granted privilege rather than an implied consequence of `view`. The Go backend SHALL mirror the action in its fallback role-permission maps (`rbac.go`/`roles.go`) for development and mock-auth parity. The Redis-cached RBAC policy (`rbacPolicyCacheKey`) SHALL be versioned when the action is introduced so the new action takes effect without waiting for the cache TTL to expire.

#### Scenario: Export action exists in the Stytch policy

- **WHEN** the Stytch RBAC policy is fetched
- **THEN** the `contact`, `deal`, and `activity` resources SHALL list `export` among their actions
- **AND** the `admin` and `manager` roles SHALL be granted `contact:export`, `deal:export`, and `activity:export`

#### Scenario: Wildcard roles resolve export

- **WHEN** a role is granted `contact:*` in the policy
- **THEN** the expanded permissions SHALL include `contact:export` because `export` is a declared action of the `contact` resource

#### Scenario: Policy cache reflects the new action after rollout

- **WHEN** the policy cache key is versioned at rollout
- **THEN** the next fetch of the policy SHALL resolve the new `export` action without waiting for the prior cache TTL

### Requirement: Role normalization

The system SHALL normalize Stytch role IDs by removing the `stytch_` prefix. For example, `stytch_member` SHALL be normalized to `member`, `stytch_admin` to `admin`.

The normalization MUST use `strings.TrimPrefix(roleID, "stytch_")` — NOT string matching on partial substrings like `"member"`.

#### Scenario: Standard role normalization

- **WHEN** a role ID `stytch_member` is received
- **THEN** it is normalized to `member`

#### Scenario: Role without prefix passes through

- **WHEN** a role ID `custom_role` is received
- **THEN** it passes through unchanged as `custom_role`

#### Scenario: Whitespace handling

- **WHEN** a role ID has leading or trailing whitespace
- **THEN** the whitespace is trimmed before prefix removal

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

### Requirement: RBACService implementation backed by Stytch policy

The `RBACService` interface SHALL have a single implementation (`StytchRBACService`) that derives all methods from the Stytch RBAC policy. The following methods MUST be supported:

| Method | Derivation |
|--------|-----------|
| `GetAllRoles()` | All roles from policy, each with `RoleInfo` including normalized ID, display name, description, and permissions |
| `GetRoleInfo(roleID)` | Single role lookup by normalized ID |
| `GetAllPermissions()` | All unique `resource:action` pairs from all role definitions |
| `GetRolePermissions(roleID)` | Delegates to `RBACPolicyService.GetRolePermissions()` |
| `GetPermissionsByCategory()` | Permissions grouped by resource (category = resource name) |
| `GetPermissionsByRoleID(roleID)` | String IDs of all permissions for a role |
| `HasPermission(roleID, permissionId)` | True if any role permission matches the permission ID |
| `GetRBACMetadata()` | Counts derived from policy (total roles, total permissions, per-role counts) |

The service SHALL be wired as the `RBACService` provided to the auth module's DI container when real Stytch credentials are present (same pattern as the `AuthProvider` fallback in `auth/cmd/init.go`). A static definitions fallback (`defaultRBACService`) SHALL be used ONLY for local development / placeholder credentials and MUST NOT be the served implementation in production. `RoleInfo.Description` SHALL be populated from the Stytch policy role description (`PolicyRole.Description`) and MUST NOT be left empty or derived from hardcoded strings when the policy defines one.

El fetch de la política (`RBACPolicyService.getPolicy` → `fetchPolicyFromStytch`) SHALL pasar por el circuit breaker de dos niveles (umbral 5, timeout 10s, half-open 2) usando el cliente con breaker de `platform/stytch` (`Client.Run`). Un fallo con breaker abierto (`ErrCircuitOpen`) o de API SHALL tratarse como política no disponible: usar la caché Redis si existe; si no, devolver vacío con log — la UI muestra "política no disponible", nunca un falso "sin permisos", y el middleware de autorización SHALL mantener el contrato 503 existente. SHALL existir exactamente UN servicio de política servido, con un único cache key versionado (`auth:stytch:rbac:policy:v2`); la implementación duplicada del paquete `platform/stytch` (cache key `stytch:rbac:policy`, retorno `[]string`) SHALL consolidarse — migrar sus consumidores al servicio consolidado y eliminar o retirar el duplicado.

#### Scenario: GetAllRoles returns roles from policy

- **WHEN** `GetAllRoles()` is called
- **THEN** it returns all roles defined in the Stytch RBAC policy
- **AND** each role has a normalized ID, display name, policy-provided description, and resolved permissions

#### Scenario: Production wiring serves the Stytch policy

- **WHEN** the backend starts with real Stytch credentials
- **AND** `GET /api/rbac/roles` is called
- **THEN** the response SHALL be derived from the Stytch RBAC policy (cached in Redis, 5-minute TTL)
- **AND** the static role definitions in code SHALL NOT be the source of the response values

#### Scenario: Policy fetch failure is distinguishable from an empty policy

- **WHEN** the Stytch RBAC policy API is unreachable and the Redis cache is empty
- **THEN** `GetAllRoles()` SHALL log the failure and return an empty result
- **AND** the authorization middleware SHALL keep returning 503 for permission checks (existing contract)
- **AND** the frontend SHALL render the "política no disponible" state for the matrix (never a false "no permissions" state)

#### Scenario: Fetch de política con circuit breaker

- **WHEN** el fetch de la política Stytch se ejecuta vía el servicio cableado
- **THEN** SHALL pasar por el circuit breaker de dos niveles (`Client.Run`, umbral 5, timeout 10s, half-open 2)
- **AND** con el breaker abierto o la API inalcanzable y caché vacía, SHALL loguear y devolver vacío (nunca un 500 hacia la UI)
- **AND** el middleware de autorización SHALL mantener el contrato 503 existente

#### Scenario: Un único servicio de política servido

- **WHEN** el backend sirve datos de roles a la aplicación
- **THEN** SHALL existir una sola implementación de servicio de política (con expansión de wildcards) con un único cache key versionado (`auth:stytch:rbac:policy:v2`)
- **AND** la implementación duplicada (`platform/stytch`, cache key `stytch:rbac:policy`) SHALL migrar sus consumidores y eliminarse o consolidarse

#### Scenario: Static fallback only in development

- **WHEN** Stytch credentials are placeholders (development mode)
- **THEN** the RBAC endpoints MAY serve static definitions as a local fallback
- **AND** the fallback SHALL be clearly gated by the placeholder-detection check (same as the auth adapter)

#### Scenario: GetRoleInfo for existing role

- **WHEN** `GetRoleInfo("admin")` is called
- **AND** the Stytch policy defines an `admin` role
- **THEN** it returns the `RoleInfo` with the correct permissions

#### Scenario: GetRoleInfo for non-existent role

- **WHEN** `GetRoleInfo("nonexistent_role")` is called
- **AND** the Stytch policy does NOT define this role
- **THEN** it returns nil

### Requirement: DTOs retained as API contract

The `RoleDTO`, `PermissionDTO`, `RolesResponse`, `PermissionsResponse`, and other API response types in `auth/rbac.go` SHALL be retained. The hardcoded *data* (`RoleInfo` variables, `AllRoles`, `AllPermissions`, `GetRoleInfo()`, `HasPermission()`) SHALL NOT be served by `GET /api/rbac/roles` nor used for authorization decisions in production; the values served SHALL come from the Stytch RBAC policy. The static definitions MAY remain in the codebase solely as a development fallback, gated by the same placeholder-credential detection as the auth adapter (`auth/cmd/init.go`), and MUST NOT be reachable in production.

#### Scenario: API response format unchanged

- **WHEN** a client calls `GET /api/rbac/roles`
- **THEN** the JSON response format SHALL be identical to the pre-change format
- **AND** the values (role names, permission lists) SHALL come from Stytch RBAC policy, not hardcoded constants

#### Scenario: Static data not served in production

- **WHEN** the backend runs with real Stytch credentials
- **THEN** `GET /api/rbac/roles` SHALL serve values derived from the Stytch RBAC policy
- **AND** the static definitions (`AllRoles`, `AllPermissions`) SHALL NOT appear in the response values

#### Scenario: Development fallback gated

- **WHEN** Stytch credentials are placeholders (development mode)
- **THEN** the static definitions MAY be served as a local fallback for the RBAC endpoints
- **AND** the fallback SHALL be gated by the same placeholder-detection check as the auth adapter

### Requirement: JWKS public keys cached locally with bounded TTL

The system SHALL cache Stytch B2B JWKS public keys locally with a TTL of at most 300 seconds, so key rotation is reflected promptly and stale keys are never accepted beyond the constitutional freshness bound.

#### Scenario: Key rotation reflected within five minutes

- **WHEN** Stytch rotates a signing key
- **THEN** the previously cached key SHALL expire within 300 seconds
- **AND** the next validation SHALL fetch the refreshed JWKS from Stytch

#### Scenario: Cache hit serves the current key

- **WHEN** a request presents a token whose key ID is cached
- **AND** the cached entry is younger than 300 seconds
- **THEN** validation SHALL use the cached key without calling the Stytch JWKS endpoint


### Requirement: La UI es ventana read-only a la política Stytch

Toda superficie de la aplicación que presente roles o permisos (incluida la página "Equipo y permisos") SHALL ser una ventana de solo lectura a la política RBAC de Stytch: la asignación miembro→rol se escribe únicamente vía la member API (Stytch), y la matriz de definiciones rol→permiso se lee de `/rbac/roles` sin edición in-app. Ninguna UI SHALL modificar directa o indirectamente la política de roles (definiciones de roles y permisos); dicha edición SHALL realizarse solo a través del Stytch dashboard o de un change futuro aprobado (`custom-roles-editor`) que respete el runtime SSOT. La fuente servida de la política (roles, descripciones, permisos efectivos con expansión de wildcards) SHALL ser el runtime Stytch, nunca definiciones estáticas en código en producción.

#### Scenario: Matriz read-only

- **WHEN** un usuario con `org:manage` abre la matriz de permisos
- **THEN** la matriz SHALL presentarse sin controles de edición de política
- **AND** los datos SHALL provenir de `/rbac/roles` (o su caché)

#### Scenario: Asignación vía member API

- **WHEN** un admin cambia el rol de un miembro
- **THEN** el cambio SHALL persistir vía la member API de Stytch (contrato existente: `Members.Update`, acción `stytch.member:update.settings.roles`)
- **AND** la política de roles definida en Stytch SHALL permanecer sin cambios por la UI
