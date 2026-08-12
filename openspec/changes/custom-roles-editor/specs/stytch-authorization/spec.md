# stytch-authorization Delta Spec

## MODIFIED Requirements

### Requirement: Escritura controlada a la política de roles

Además de la ventana read-only de `equipo-permisos`, el sistema SHALL permitir la edición controlada de roles personalizados escribiendo en la política RBAC de Stytch a través de su API de roles. Las escrituras SHALL: (a) pasar por el circuito breaker de Stytch (abierto → rechazo sin efecto); (b) ser idempotentes; (c) proteger roles del sistema; (d) registrarse en auditoría. La política de Stytch SHALL permanecer como el único SSOT de definiciones de roles; la UI no mantiene ninguna copia editable de la política.

#### Scenario: Escritura a Stytch con breaker

- **WHEN** se guarda un rol personalizado
- **THEN** la escritura SHALL pasar por el breaker y la API de roles de Stytch
- **AND** con breaker abierto SHALL rechazarse sin escritura local

#### Scenario: Sin copia local editable

- **WHEN** un usuario abre el editor de roles
- **THEN** los datos SHALL provenir de la lectura de la política (caché TTL 5 min)
- **AND** SHALL NOT existir una tabla o modelo local que duplique la política

### Requirement: Permiso roles:manage

La política SHALL definir el permiso `roles:manage`, asignado al rol admin, requerido para las operaciones de escritura del editor. El gate de la vista `?view=roles` SHALL combinar `org:manage` + `roles:manage`.

#### Scenario: Sin roles:manage

- **WHEN** un usuario sin `roles:manage` intenta abrir el editor
- **THEN** la vista SHALL mostrar 403 sin controles de edición
