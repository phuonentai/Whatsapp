# stytch-authorization Delta Spec

## MODIFIED Requirements

### Requirement: Permisos de bandeja independientes de org:manage

La política RBAC de Stytch SHALL definir permisos de bandeja que permitan a miembros no-admin leer conversaciones y responder manualmente: `inbox:view` (leer lista + thread) y `inbox:reply` (enviar mensaje manual). La asignación de estos permisos a roles (miembro, agente, admin) SHALL definirse en la política de Stytch (runtime SSOT). El enforcement SHALL ser server-side en los endpoints de conversaciones y mensajes: lectura/escritura manual con `inbox:view`/`inbox:reply`; cerrar/reabrir, quick replies, sugerencias IA y writing assist requieren `org:manage`.

#### Scenario: Miembro lee y responde

- **WHEN** un miembro con `inbox:view` e `inbox:reply` abre la bandeja
- **THEN** SHALL poder listar conversaciones, abrir threads y enviar respuestas manuales

#### Scenario: Miembro sin permiso de administración

- **WHEN** un miembro sin `org:manage` intenta cerrar/reabrir o usar quick replies/sugerencias
- **THEN** la UI SHALL ocultar esos controles
- **AND** el backend SHALL rechazar la operación (403) si se intenta directo

#### Scenario: Rollback de política

- **WHEN** se revierte este change
- **THEN** la política de Stytch SHALL restaurar el gate `org:manage` completo de la bandeja
- **AND** el repositorio SHALL revertirse a la versión previa (rollback dual Git + Stytch)
