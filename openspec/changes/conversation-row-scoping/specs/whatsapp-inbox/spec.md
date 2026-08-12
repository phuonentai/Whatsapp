# whatsapp-inbox Delta Spec

## ADDED Requirements

### Requirement: Lista de conversaciones acotada por scope de miembro

La lista de conversaciones SHALL acotarse por el scope del miembro (regla de unión): `assignee_stytch_member_id = miembro`, O el contacto pertenece a una empresa con `owner_account_id` del miembro, O el miembro tiene `inbox:view_all` (todas), O la conversación está sin asignar y el miembro tiene `inbox:view_unassigned` (cola). El predicado SHALL aplicarse server-side en `GET /crm/conversaciones` y en todas las consultas derivadas (poll, unread, paginación).

#### Scenario: Miembro con asignación directa ve sus chats

- **WHEN** un miembro tiene `inbox:view` y una conversación tiene `assignee_stytch_member_id` = su `stytch_member_id`
- **THEN** la conversación SHALL aparecer en su lista

#### Scenario: Owner de empresa ve chats de sus cuentas

- **WHEN** un miembro es `owner_account_id` de una `crm.companies` y el contacto de una conversación tiene `company_id` = esa empresa
- **THEN** la conversación SHALL aparecer en su lista aunque no sea el assignee

#### Scenario: view_all ve todo

- **WHEN** un miembro tiene `inbox:view_all` y `inbox:view`
- **THEN** SHALL ver todas las conversaciones de la organización

#### Scenario: view_unassigned ve solo la cola

- **WHEN** un miembro tiene `inbox:view_unassigned` (sin `inbox:view_all`)
- **THEN** SHALL ver solo conversaciones con `assignee_stytch_member_id IS NULL`
- **AND** una conversación asignada a otro miembro SHALL NO ser visible

#### Scenario: Conversación fuera de scope no filtra al thread

- **WHEN** un miembro navega a una conversación fuera de su scope (URL directa o id en payload)
- **THEN** el backend SHALL devolver 404 (no encontrada, no 403) sin filtrar existencia

### Requirement: Unread y métricas de lista respetan el scope

El conteo de no leídos, el poll de 5s y cualquier métrica derivada de la lista SHALL aplicar el mismo predicado de scope que la lista, para que no existan indicadores fantasma de conversaciones invisibles.

#### Scenario: Unread sin fugas

- **WHEN** un miembro sin `inbox:view_all` consulta sus no leídos
- **THEN** el conteo SHALL incluir solo conversaciones dentro de su scope
- **AND** SHALL NO mostrar indicadores de conversaciones que no puede abrir

#### Scenario: Poll respeta scope

- **WHEN** el poll de 5s refresca la lista/thread de un miembro
- **THEN** las filas devueltas SHALL ser las mismas que la consulta inicial de lista (mismo predicado)

### Requirement: Re-asignación de conversaciones (endpoint)

El endpoint de re-asignación (`PATCH /crm/conversaciones/:id/assignee`) SHALL exigir el permiso `inbox:reassign`, validar que el destino pertenece al mismo org (directorio Stytch, ver stytch-authorization) y registrar actor/origen/destino en el audit ledger. Fuera de scope → 404 (sin filtrar existencia); sin permiso → 403; directorio no disponible → 503 `member_directory_unavailable`.

#### Scenario: Re-asignación válida

- **WHEN** un miembro con `inbox:reassign` re-asigna una conversación visible a un miembro activo del mismo org
- **THEN** la conversación SHALL actualizar su `assignee_stytch_member_id`
- **AND** el evento SHALL registrarse en el audit ledger con actor, origen y destino

#### Scenario: Sin permiso → 403

- **WHEN** un miembro sin `inbox:reassign` intenta re-asignar
- **THEN** el endpoint SHALL responder 403
- **AND** la operación SHALL NO ejecutarse

#### Scenario: Fuera de scope → 404

- **WHEN** un miembro con `inbox:reassign` intenta re-asignar una conversación fuera de su scope
- **THEN** el endpoint SHALL responder 404 (no encontrada, sin filtrar existencia)

#### Scenario: Directorio no disponible → 503

- **WHEN** el circuito del directorio de miembros está abierto o la cache está vacía con API inalcanzable
- **THEN** el endpoint SHALL responder 503 `member_directory_unavailable`
- **AND** la bandeja (lectura/respuesta) SHALL permanecer funcional
