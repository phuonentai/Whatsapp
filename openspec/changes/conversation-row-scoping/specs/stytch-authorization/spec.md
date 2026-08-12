# stytch-authorization Delta Spec

## ADDED Requirements

### Requirement: Permisos de scope de conversaciones en la política Stytch

La política RBAC de Stytch SHALL definir permisos de scope de bandeja independientes de `org:manage`: `inbox:view_all` (ver todas las conversaciones de la organización), `inbox:view_unassigned` (ver la cola de conversaciones sin asignar) e `inbox:reassign` (re-asignar una conversación a otro miembro). La asignación de estos permisos a roles SHALL definirse en la política de Stytch (runtime SSOT; patrón recurso:acción, composable: los miembros con múltiples roles acumulan la unión de permisos). El enforcement SHALL ser server-side.

#### Scenario: view_all concede scope org-wide sin acciones destructivas

- **WHEN** un miembro tiene `inbox:view_all` pero no `org:manage`
- **THEN** SHALL ver todas las conversaciones de la organización (scope org-wide)
- **AND** SHALL poder usar solo las acciones que sus permisos de acción concedan (p. ej. `inbox:reply`, `inbox:reassign`)
- **AND** SHALL NO poder cerrar/reabrir conversaciones ni usar quick replies/sugerencias si no tiene `org:manage`

#### Scenario: view_unassigned concede la cola

- **WHEN** un miembro tiene `inbox:view_unassigned`
- **THEN** SHALL ver las conversaciones con `assignee_stytch_member_id IS NULL`
- **AND** una conversación asignada SHALL NO ser visible para él solo por este permiso

#### Scenario: reassign concede re-asignación

- **WHEN** un miembro tiene `inbox:reassign` y ve una conversación
- **THEN** SHALL poder cambiar su `assignee_stytch_member_id` a otro miembro de la organización
- **AND** el cambio SHALL registrarse en el audit ledger con actor, origen y destino

#### Scenario: Miembro sin permisos de scope

- **WHEN** un miembro tiene `inbox:view`/`inbox:reply` pero ninguno de `inbox:view_all`, `inbox:view_unassigned` ni ownership sobre la conversación
- **THEN** la conversación SHALL NO ser visible (el query devuelve cero filas)

#### Scenario: Rollback de política

- **WHEN** se revierte este change
- **THEN** la política de Stytch SHALL remover `inbox:view_all`, `inbox:view_unassigned` e `inbox:reassign`
- **AND** el repositorio SHALL revertirse a la versión previa (rollback dual Git + Stytch)

### Requirement: Precedencia de autorización — política cacheada como fuente runtime

El sistema SHALL resolver permisos de scope desde la **política Stytch cacheada** (Redis, TTL 5 min; 503 Service Unavailable si la API de Stytch está inalcanzable y la cache vacía — spec vivo `stytch-authorization`). El mapeo rol→scope tipado (structs Go en domain) SHALL ser **tipos de compilación + fallback dev/mock** (precedente `internal/modules/auth/rbac.go`/`roles.go`), NUNCA la fuente runtime de decisiones de autorización.

#### Scenario: El middleware resuelve desde la política cacheada

- **WHEN** un request llega al backend
- **THEN** el middleware SHALL resolver permisos desde la política Stytch cacheada (fuente runtime)
- **AND** SHALL escribir las session vars de scope correspondientes (`app.current_member_id`, `app.is_view_all`, `app.is_view_unassigned`) con `SET LOCAL` en la transacción del request
- **AND** el contrato tipado Go SHALL usarse solo como tipos de compilación y fallback dev/mock

#### Scenario: Permisos nuevos espejados en el fallback dev/mock

- **WHEN** se añaden `inbox:view_all`, `inbox:view_unassigned` e `inbox:reassign` a la política Stytch
- **THEN** SHALL espejarse en los maps de fallback `internal/modules/auth/rbac.go`/`roles.go` (paridad mock-auth)
- **AND** la cache key de la política SHALL versionarse (patrón `auth:stytch:rbac:policy:v2`) para que los permisos nuevos tomen efecto sin esperar el TTL

#### Scenario: Composición supervisor

- **WHEN** un rol combina `inbox:view_all` con `inbox:reply`/`inbox:reassign` sin `org:manage`
- **THEN** el sistema SHALL tratar al miembro como supervisor: ve todo, responde y reasigna, pero no cierra ni gestiona configuración

### Requirement: Directorio de miembros para reasignación (Stytch Members API)

El sistema SHALL obtener la lista de miembros asignables del directorio de la organización desde la **Stytch B2B Members API** (contrato validado en docs oficiales: `POST /v1/b2b/organizations/members/search` con `organization_ids` del org solicitante, query vacía, paginación por `next_cursor`, filtro `statuses: [active]`; SDK Go `stytch/b2b/organizations/members`). La llamada SHALL envolverse en el circuit-breaker de dos niveles para outbound a Stytch (umbral 5, timeout 10s, probe half-open 2) + cache Redis (TTL 5 min, patrón de la política RBAC). Solo se persiste/retorna `stytch_member_id` (FK lógico; nunca credenciales ni datos de sesión).

#### Scenario: Picker lista miembros del org

- **WHEN** un miembro con `inbox:reassign` abre el picker de asignación
- **THEN** la lista SHALL provenir del directorio (Members search del org, cacheado, con circuit-breaker)
- **AND** SHALL incluir solo miembros `active` del mismo org

#### Scenario: Directorio no disponible — degradación visible

- **WHEN** el circuit-breaker está abierto o la cache está vacía y la API de Stytch inalcanzable
- **THEN** el picker SHALL ocultarse con estado de retry (sin ghost)
- **AND** el endpoint de re-asignación SHALL responder 503 `member_directory_unavailable`
- **AND** la lectura/respuesta de la bandeja SHALL permanecer funcional

#### Scenario: Destino validado en el mismo org

- **WHEN** un miembro con `inbox:reassign` re-asigna a un miembro destino
- **THEN** el destino SHALL pertenecer al mismo org del solicitante (validación server-side contra el directorio)
- **AND** SHALL NO aceptarse un `stytch_member_id` de otro org
