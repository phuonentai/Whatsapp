# custom-roles-editor Delta Spec

## ADDED Requirements

### Requirement: Editor in-app de roles personalizados

El sistema SHALL ofrecer un editor de roles personalizados (vista `?view=roles` o tab avanzado de `access`, gate `org:manage` + permiso `roles:manage`) que permita crear, editar y archivar roles con: nombre, descripción y conjunto de permisos seleccionados del catálogo (metadata de `/rbac/roles`). La validación SHALL rechazar permisos desconocidos, wildcards sin expansión validada, nombres duplicados y la edición de roles del sistema. La escritura SHALL realizarse en la política de Stytch vía su API (runtime SSOT), nunca en una copia local.

#### Scenario: Crear rol personalizado

- **WHEN** un admin con `roles:manage` crea un rol personalizado con permisos válidos
- **THEN** el rol SHALL persistirse en la política de Stytch
- **AND** SHALL aparecer en la matriz de permisos y en el role select de asignación

#### Scenario: Rol del sistema protegido

- **WHEN** un usuario intenta editar o eliminar un rol del sistema (`member`/`approver`/`admin`)
- **THEN** la UI SHALL bloquear la operación
- **AND** el backend SHALL rechazarla (403)

#### Scenario: Validación rechaza entrada inválida

- **WHEN** se envía un permiso desconocido o un nombre duplicado
- **THEN** el editor SHALL mostrar error inline
- **AND** SHALL NOT llamar a la API de Stytch

### Requirement: Escritura idempotente con breaker y auditoría

Toda escritura a la política SHALL pasar por el circuito de Stytch (breaker de dos niveles; si abierto → rechazo sin escritura local) y SHALL ser idempotente (reintentos seguros). Cada cambio de política (crear/editar/archivar) SHALL registrarse en el audit ledger con actor, timestamp y diff de permisos. Tras guardar, la UI SHALL mostrar la nota "Los cambios aplican en hasta 5 minutos".

#### Scenario: Breaker abierto

- **WHEN** el circuito de Stytch está abierto durante una escritura de rol
- **THEN** la operación SHALL fallar con error
- **AND** SHALL NOT realizarse escritura local ni parcial en la política

#### Scenario: Cambio auditado

- **WHEN** un rol se crea, edita o archiva
- **THEN** SHALL registrarse en el audit con actor, timestamp y diff
- **AND** la UI SHALL mostrar la nota de propagación (hasta 5 minutos)

### Requirement: Rollback de política documentado

El change SHALL documentar el procedimiento de rollback de la política de roles de Stytch (backup/versionado previo a cada escritura y restauración) junto con el revert de Git, para que ambos SSOT puedan revertirse en fase.

#### Scenario: Rollback dual

- **WHEN** se revierte este change
- **THEN** la política de roles de Stytch SHALL restaurarse al estado previo (procedimiento documentado)
- **AND** el repositorio SHALL revertirse en fase
