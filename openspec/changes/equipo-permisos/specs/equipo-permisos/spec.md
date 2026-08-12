# equipo-permisos Delta Spec

## ADDED Requirements

### Requirement: Página consolidada de derechos con tres capas

El sistema SHALL ofrecer una vista consolidada de gestión de derechos (`/dashboard/settings?view=access`) accesible solo para usuarios con `org:manage`, con tres tabs URL-addressables (`?tab=members|matrix|modules`):

1. **Miembros**: lista de miembros con asignación de rol (select con descripción del rol), invitar, remover — reutilizando la lógica existente de `MemberList`/`InviteMember` (ConfirmDialog, last-admin guard, self-protection). Las opciones del selector de rol y sus descripciones SHALL provenir de la misma fuente que la matriz (`/rbac/roles`), con fallback al copy en español (`lib/copy/ui.ts`); las descripciones de rol hardcodeadas en inglés en el componente NO SHALL ser la fuente primaria. Tras un cambio de rol SHALL mostrarse una nota inline "Los cambios aplican en hasta 5 minutos".
2. **Matriz de permisos**: tabla read-only de roles × recursos, fuente `/rbac/roles`, con celdas ✓/parcial/— y Tooltip que explique qué rol otorga el permiso (incluida expansión de wildcards a las acciones declaradas); filtro por recurso; columna admin con ancla visual.
3. **Módulos**: resumen de módulos activos con badge de fuente de plan, sin toggles duplicados; links a `?view=modules` (toggles) y `?view=subscription` (upgrade).

La vista NO SHALL permitir editar la política RBAC: la matriz es de solo lectura y el origen de verdad es la política Stytch (runtime SSOT). La vista SHALL registrarse en el allowlist de gates existente de `settings-content.tsx` (mismo mecanismo que `?view=members`/`?view=modules`), gateado por `org:manage`; una vista no registrada SHALL caer al overview, nunca renderizar datos sin gate.

#### Scenario: Admin abre la vista consolidada

- **WHEN** un usuario con `org:manage` abre `/dashboard/settings?view=access`
- **THEN** SHALL renderizar los tres tabs con miembros, matriz read-only y módulos
- **AND** el tab activo SHALL reflejar el parámetro `tab` de la URL

#### Scenario: Vista registrada en el allowlist de gates

- **WHEN** el estado de permisos está listo y `?view=access` está en la URL
- **THEN** la vista SHALL resolverse solo si el usuario tiene `org:manage`
- **AND** sin `org:manage` SHALL renderizar la vista 403 sin datos de matriz ni de miembros

#### Scenario: Cambio de rol con nota de propagación

- **WHEN** un admin cambia el rol de un miembro
- **THEN** la asignación SHALL persistir vía la member API existente
- **AND** SHALL mostrarse la nota "Los cambios aplican en hasta 5 minutos"

#### Scenario: Último admin protegido

- **WHEN** se intenta remover o degradar al último admin
- **THEN** la vista SHALL mostrar un error inline (o el confirm de dos pasos para auto-degradación) sin permitir la operación

#### Scenario: Usuario sin permiso

- **WHEN** un usuario sin `org:manage` abre `/dashboard/settings?view=access`
- **THEN** SHALL renderizar una vista 403 sin datos de la matriz ni de miembros

### Requirement: Matriz de permisos con metadata y expansión de wildcards

La matriz SHALL presentar los roles y permisos efectivos devueltos por `/rbac/roles` con nombres de display (`displayName`), descripciones y categorías; las celdas SHALL usar ✓/parcial/— con texto, nunca color-only; cada celda SHALL ofrecer un Tooltip que explique el origen (qué rol y qué `resource:action`, incluyendo expansión de wildcards como `contact:*` → todas las acciones declaradas). La matriz SHALL incluir filtro por recurso y NO SHALL mostrar IDs crudos como etiquetas primarias (IDs solo en tooltips). La matriz SHALL consultar roles con `staleTime` a lo sumo igual a la TTL de la caché de política (5 minutos), SHALL ofrecer refetch manual, y SHALL distinguir "sin permisos" de "política no disponible": si `/rbac/roles` devuelve una lista vacía, la matriz SHALL renderizar un estado de indisponibilidad con retry (nunca un vacío de permisos). Si un permiso wildcard referencia un recurso no definido en la política, la celda SHALL mostrar el wildcard literal con una nota de permiso amplio.

#### Scenario: Celda con explicación

- **WHEN** el usuario posa el foco sobre una celda de la matriz
- **THEN** SHALL aparecer un tooltip con el `resource:action`, el rol que lo otorga y su displayName
- **AND** si el permiso proviene de un wildcard, el tooltip SHALL listar las acciones efectivas
- **AND** si el recurso del wildcard no está definido en la política, el tooltip SHALL indicar permiso amplio (wildcard literal)

#### Scenario: Filtro por recurso

- **WHEN** el usuario filtra por recurso
- **THEN** la matriz SHALL mostrar solo las filas/columnas del recurso seleccionado

#### Scenario: Frescura limitada a la TTL de la política

- **WHEN** la matriz ya cargó roles dentro de los últimos 5 minutos
- **THEN** SHALL reutilizar los datos en caché (sin refetch automático adicional)
- **AND** el usuario SHALL poder forzar un refetch manual con un control visible

#### Scenario: Lista de roles vacía ≠ sin permisos

- **WHEN** `/rbac/roles` responde con una lista vacía de roles
- **THEN** la matriz SHALL renderizar "política no disponible" con opción de reintentar
- **AND** NO SHALL renderizar una matriz vacía que sugiera ausencia de permisos

### Requirement: Preview de impacto por miembro

La vista SHALL permitir seleccionar un miembro y mostrar qué superficies podrá usar según sus permisos efectivos (p. ej. bandeja: responder y aprobar sugerencias IA; knowledge: chatear sin gestionar documentos; AI Copilot: solo lectura; Módulos: oculto), generado desde la misma lógica de gating de la navegación (sidebar y gates de vista) para que no pueda desviarse de la aplicación real. El preview SHALL ser un disclosure accesible (aria-expanded) con foco dentro al abrir.

#### Scenario: Preview de un miembro

- **WHEN** el admin selecciona un miembro
- **THEN** SHALL renderizar el resumen de superficies con ✓/✗ según sus permisos efectivos
- **AND** el resumen SHALL derivarse de la misma fuente que la navegación real

### Requirement: Auditoría de cambios de rol visible

La vista SHALL mostrar una lista compacta de cambios recientes de rol (quién cambió qué rol a quién, cuándo) y un enlace al audit log completo, provenientes del ledger de auditoría existente. La sección de cambios recientes SHALL gatearse por `audit:view` (mismo predicado que `audit-log-view`) además de `org:manage`: un usuario sin `audit:view` NO SHALL ver la lista ni disparar su fetch, y el enlace al audit log SHALL permanecer funcional solo para quien tiene `audit:view`.

#### Scenario: Cambios recientes inline

- **WHEN** un usuario con `org:manage` Y `audit:view` abre la vista consolidada y existen cambios de rol registrados
- **THEN** SHALL mostrarse la lista compacta con actor, sujeto, rol y fecha
- **AND** el enlace SHALL navegar a la vista de audit log completa

#### Scenario: Sin audit:view no hay ledger inline

- **WHEN** un usuario con `org:manage` pero sin `audit:view` abre la vista consolidada
- **THEN** la sección de cambios recientes SHALL ocultarse sin disparar fetch del ledger
- **AND** no SHALL renderizarse ningún dato de auditoría en la vista
