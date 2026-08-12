# inbox-ui Delta Spec

## ADDED Requirements

### Requirement: Tabs de visibilidad en la bandeja

La bandeja SHALL ofrecer vistas de scope según los permisos del miembro: "Mis chats" (assignee o owner de empresa = miembro), "Cola sin asignar" (visible solo con `inbox:view_unassigned`) y "Todos" (visible solo con `inbox:view_all` u `org:manage`). Las vistas SHALL derivarse del mismo predicado que el backend (sin filtrado client-side que difiera).

#### Scenario: Miembro con asignaciones ve "Mis chats"

- **WHEN** un miembro con `inbox:view` abre la bandeja
- **THEN** SHALL ver la vista "Mis chats" con las conversaciones de su scope (assignee ∪ owner de empresa)
- **AND** SHALL NO ver conversaciones de otros miembros ni de la cola sin permiso

#### Scenario: view_unassigned ve la cola

- **WHEN** un miembro con `inbox:view_unassigned` selecciona "Cola sin asignar"
- **THEN** SHALL ver las conversaciones con `assignee_stytch_member_id IS NULL`

#### Scenario: view_all ve "Todos"

- **WHEN** un miembro con `inbox:view_all` (u `org:manage`) selecciona "Todos"
- **THEN** SHALL ver todas las conversaciones de la organización

#### Scenario: Tabs ausentes sin permiso

- **WHEN** un miembro no tiene `inbox:view_unassigned` ni `inbox:view_all`
- **THEN** los tabs "Cola sin asignar" y "Todos" SHALL NO mostrarse

### Requirement: Asignación y re-asignación de conversaciones

El sistema SHALL permitir asignar/re-asignar una conversación a un miembro de la organización desde la cabecera del thread, solo con el permiso `inbox:reassign`. La acción SHALL persistirse server-side (`assignee_stytch_member_id`) y registrarse en el audit ledger con actor, origen y destino.

#### Scenario: Re-asignación con permiso

- **WHEN** un miembro con `inbox:reassign` re-asigna una conversación visible a otro miembro
- **THEN** la conversación SHALL actualizar su `assignee_stytch_member_id`
- **AND** el evento SHALL registrarse en el audit ledger

#### Scenario: Sin permiso no hay picker

- **WHEN** un miembro sin `inbox:reassign` ve una conversación
- **THEN** el picker de asignación SHALL NO mostrarse (hide, no ghost)
- **AND** el backend SHALL rechazar la operación (403) si se intenta directo

#### Scenario: Reclamar desde la cola

- **WHEN** un miembro con `inbox:view_unassigned` y `inbox:reassign` se auto-asigna una conversación de la cola
- **THEN** la conversación SHALL pasar a su scope ("Mis chats")
- **AND** SHALL salir de la cola para el resto de miembros con `inbox:view_unassigned`

#### Scenario: Directorio de miembros no disponible — picker con retry

- **WHEN** un miembro con `inbox:reassign` abre el picker y el directorio de miembros no está disponible (circuit abierto o cache vacía con API inalcanzable)
- **THEN** el picker SHALL mostrarse en estado de no-disponible con opción de reintentar (hide funcional, sin ghost ni opciones falsas)
- **AND** el thread y el composer SHALL permanecer operativos

#### Scenario: Free tier sin controles de scope

- **WHEN** una organización en free tier (flag `conversation_row_scoping` false) abre la bandeja
- **THEN** SHALL NO mostrar tabs de cola ni pickers de asignación
- **AND** la lista SHALL mostrar todas las conversaciones (org-scope)
