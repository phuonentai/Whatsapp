# stytch-authorization Delta Spec

## MODIFIED Requirements

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

#### Scenario: GetRoleInfo for existing role

- **WHEN** `GetRoleInfo("admin")` is called
- **AND** the Stytch policy defines an `admin` role
- **THEN** it returns the `RoleInfo` with the correct permissions

#### Scenario: GetRoleInfo for non-existent role

- **WHEN** `GetRoleInfo("nonexistent_role")` is called
- **AND** the Stytch policy does NOT define this role
- **THEN** it returns nil

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

## ADDED Requirements

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
