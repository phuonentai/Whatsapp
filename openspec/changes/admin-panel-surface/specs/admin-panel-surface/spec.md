# admin-panel-surface Delta Spec

## ADDED Requirements

### Requirement: Shell de operador con gate de plataforma

El sistema SHALL ofrecer un shell de operador (ruta dedicada, fuera del layout de `/dashboard`) con sidebar de plataforma: Organizaciones, Uso IA, Auditoría (SIN Siigo). El acceso SHALL requerir el permiso `platform:operate` (rol `platform_admin` en la política Stytch, org de plataforma dedicada); sin él, el shell SHALL mostrar 403 y SHALL NOT estar enlazado en la navegación de miembros. El shell SHALL usar el lenguaje visual del diseño (mismas superficies y componentes que el producto).

#### Scenario: Operador accede al shell

- **WHEN** un usuario con `platform:operate` abre la ruta de plataforma
- **THEN** SHALL renderizar el shell con las secciones de operador
- **AND** la navegación SHALL marcar la sección activa (aria-current)

#### Scenario: Miembro sin permiso

- **WHEN** un miembro normal intenta abrir la ruta de plataforma
- **THEN** SHALL mostrar 403 sin datos
- **AND** la ruta SHALL NOT aparecer en la navegación de `/dashboard`

#### Scenario: Política RBAC no disponible

- **WHEN** el servicio de política Stytch no responde y la caché Redis está vacía
- **THEN** los endpoints de plataforma SHALL devolver 503 (nunca un falso allow ni un falso 403)
- **AND** la UI SHALL mostrar "política no disponible"

### Requirement: Contexto de plataforma (request-context model)

Las rutas de plataforma (`/api/v1/platform/*`) SHALL autenticar al operador con la sesión Stytch existente (JWT verificado por JWKS en edge / `X-Forwarded-Auth`) y SHALL resolver el permiso `platform:operate` desde la política RBAC cacheada (cache Redis 5 min) más los roles del miembro (claim `roles` del JWT), sin llamada Stytch por request. El contexto de plataforma SHALL ser un principal de plataforma (identidad del operador + org de plataforma dedicada); NO se usa `authorization_check` de Stytch para lecturas cross-org (acopla el check a la org de la sesión). Los endpoints member-scoped SHALL permanecer org-scoped y SHALL NOT exponer datos cross-org.

#### Scenario: Filtro org_id validado

- **WHEN** un operador consulta con un `org_id` que no existe en `organizations`
- **THEN** SHALL devolver 404 sin datos
- **AND** SHALL NOT usar el `org_id` como mecanismo de scoping derivado del llamante

#### Scenario: Regresión de scoping por org

- **WHEN** un operador de plataforma llama a un endpoint member-scoped existente
- **THEN** el endpoint SHALL resolver el `OrganizationID` del contexto del miembro (org `platform-ops` del operador), no el del filtro de plataforma
- **AND** SHALL NOT exponer datos de otra organización

#### Scenario: Fallback de política

- **WHEN** la política RBAC no está disponible y la caché está vacía
- **THEN** los endpoints de plataforma SHALL devolver 503 conforme al contrato `stytch-authorization`

### Requirement: Vista Organizaciones cross-org

El shell SHALL listar organizaciones con búsqueda y paginación: nombre, conteo de miembros, plan/estado de suscripción, estado de conexión (WhatsApp/Instagram — NO Siigo). El detalle de org SHALL mostrar el estado de sus integraciones y uso IA. La vista SHALL ser read-only en v1 y SHALL NOT exponer datos de CRM/contactos/conversaciones (purpose limitation Ley 1581).

#### Scenario: Lista de orgs

- **WHEN** un operador abre Organizaciones
- **THEN** SHALL mostrar la tabla de orgs con estado de suscripción y conexiones
- **AND** la búsqueda y paginación SHALL filtrar/limitar los resultados

#### Scenario: Detalle sin datos de negocio

- **WHEN** un operador abre el detalle de una org
- **THEN** SHALL mostrar SOLO estado de integraciones (WhatsApp/Instagram) y uso IA
- **AND** SHALL NOT incluir actividad CRM, contactos ni contenido de conversaciones

### Requirement: Vista Uso IA por org (plataforma)

El shell SHALL mostrar el uso de IA por org y periodo desde el ledger `ai_usage`: tokens input/output/embedding, créditos usados, límite `ai_credits_max` y % de uso. SHALL incluir tablas de tasa de modelo (modelo, feature, precio/token) como referencia read-only y filtros por periodo/org. Las agregaciones platform-wide SHALL respetar límites de paginación server-side y SHALL validar cobertura de índices (period-first) antes de producción (spike); sin datos → 0 o "—".

#### Scenario: Tabla de uso por org

- **WHEN** un operador abre Uso IA
- **THEN** SHALL mostrar por org los tokens y créditos del periodo
- **AND** SHALL calcular el % frente al límite del plan

#### Scenario: Sin datos de un periodo

- **WHEN** un org no tiene uso en el periodo
- **THEN** SHALL mostrarse 0 o "—" sin fabricar valores

### Requirement: Siigo fuera de alcance de la plataforma

El sistema SHALL NOT exponer ningún dato de Siigo en la superficie de plataforma: ni sección en el shell, ni estado de conexión, ni credenciales, ni datos de facturación/numeración/import. La superficie Siigo del tenant (`siigo-admin-view` en settings, gate rol `admin`, endpoints org-scoped) SHALL permanecer sin cambios. Ninguna ruta `/api/v1/platform/*` SHALL consultar tablas o campos de Siigo.

#### Scenario: Sin sección Siigo en el shell

- **WHEN** un operador abre el shell de plataforma
- **THEN** SHALL NOT existir sección ni enlace Siigo
- **AND** la navegación de plataforma SHALL contener solo Organizaciones, Uso IA y Auditoría

#### Scenario: Sin datos Siigo en respuestas de plataforma

- **WHEN** un operador consulta cualquier endpoint de plataforma
- **THEN** la respuesta SHALL NOT contener campos Siigo (estado de conexión, credenciales, facturación, numeración)
- **AND** SHALL NOT consultarse tablas ni queries de Siigo

#### Scenario: Siigo del tenant intacto

- **WHEN** un admin de org abre settings con `?view=siigo-admin`
- **THEN** la vista Siigo SHALL seguir funcionando sin cambios (misma lógica, mismo gate, mismos endpoints org-scoped)

### Requirement: Auditoría cross-org

El shell SHALL exponer los eventos del ledger (`ai_usage_events`, append-only) y la actividad operativa de plataforma con filtros (org, tipo, fecha). La vista SHALL ser read-only, SHALL NOT permitir mutar eventos y SHALL NOT incluir actividad CRM, contactos ni contenido de conversaciones en v1.

#### Scenario: Filtros de auditoría

- **WHEN** un operador filtra por org/tipo/fecha
- **THEN** SHALL mostrarse solo los eventos que coinciden
- **AND** los eventos SHALL permanecer inmutables

#### Scenario: Exclusión de datos de clientes

- **WHEN** un operador consulta la auditoría de una org
- **THEN** SHALL listarse SOLO eventos del ledger `ai_usage_events` y eventos operativos
- **AND** SHALL NOT aparecer notas, llamadas, correos, mensajes de WhatsApp, contactos ni contenido de conversaciones

### Requirement: Auditoría de acceso de plataforma

El sistema SHALL registrar toda lectura cross-org de la plataforma (listado, búsqueda, filtro, detalle) en una tabla aditiva append-only `platform_access_log`: `actor_stytch_member_id`, `actor_stytch_organization_id`, `target_organization_id` (nullable), `action`, `created_at`. La retención SHALL ser 90 días (configurable). La tabla SHALL NOT almacenar datos de negocio ni contenido de la org objetivo.

#### Scenario: Lectura registrada

- **WHEN** un operador lee datos cross-org (listado o detalle)
- **THEN** SHALL insertarse una fila en `platform_access_log` con actor, org objetivo y acción

#### Scenario: Sin datos de negocio en el log

- **WHEN** se registra un acceso
- **THEN** SHALL persistirse solo metadatos de acceso (actor, org, acción, timestamp)
- **AND** SHALL NOT persistirse datos de CRM, contactos ni contenido de conversaciones
