# lean-data-isolation Delta Spec

## ADDED Requirements

### Requirement: RLS a nivel miembro con session vars de scope (semántica transaccional)

El sistema SHALL introducir la primera implementación real de RLS (el spec vivo define el patrón opt-in; ninguna migración lo implementa hoy) extendiéndolo a nivel miembro en `crm.conversations`. El middleware SHALL setear las session vars `app.current_member_id` (stytch_member_id del miembro), `app.is_view_all` (bool) y `app.is_view_unassigned` (bool) — derivadas del contrato rol→scope con la política Stytch cacheada como fuente runtime — **exclusivamente con `SET LOCAL` dentro de la transacción del request** (nunca `SET` a nivel sesión sobre conexiones del pool de database/sql, para que una conexión reutilizada no herede scope de otro miembro). La política RLS SHALL aplicar el predicado de unión: `organization_id = app.current_organization_id` AND ( `app.is_view_all` OR `assignee_stytch_member_id = app.current_member_id` OR empresa del contacto con owner = miembro OR (`assignee IS NULL` AND `app.is_view_unassigned`) ).

#### Scenario: Política RLS bloquea filas fuera de scope

- **WHEN** una consulta se ejecuta con RLS activa y session vars de scope de un miembro (seteadas con `SET LOCAL` en la transacción del request)
- **THEN** PostgreSQL SHALL devolver solo filas dentro de su scope
- **AND** filas fuera de scope SHALL ser invisibles (cero filas)

#### Scenario: Reuso de conexión no filtra scope entre miembros

- **WHEN** dos requests de miembros distintos se ejecutan en la misma conexión del pool en secuencia
- **THEN** el primer request SHALL usar sus propias vars de scope dentro de su transacción (`SET LOCAL`)
- **AND** el segundo request SHALL partir sin vars heredadas (estado limpio al abrir su transacción)

#### Scenario: view_all materializado como session var

- **WHEN** el middleware resuelve que el miembro tiene `inbox:view_all`
- **THEN** SHALL setear `app.is_view_all = true` dentro de la transacción
- **AND** la política SHALL permitir todas las filas de la organización

#### Scenario: RLS sin session vars retorna cero filas (fail-closed)

- **WHEN** una consulta interactiva se ejecuta contra `crm.conversations` con RLS activa sin session vars de miembro en la transacción
- **THEN** PostgreSQL SHALL devolver cero filas
- **AND** el sistema SHALL registrar una anomalía observable (métrica/log de política activa con vars ausentes) — el path interactivo SHALL NO devolver lista vacía silenciosa

### Requirement: Ingestión de webhook fuera del scope (bypass controlado)

Los INSERT de `crm.conversations` y `crm.contacts` originados por el webhook de WhatsApp SHALL ejecutarse con el rol `app_session` (bypass RLS) o con una política RLS que permita explícitamente INSERT del servicio de webhook, de modo que el inbound nunca sea bloqueado por el scope. Los UPDATE/DELETE del webhook SHALL limitarse a metadata de sistema (p. ej. `whatsapp_message_id`, estado de entrega) y SHALL NO poder re-asignar ni borrar conversaciones.

#### Scenario: Inbound crea conversación con RLS activa

- **WHEN** llega un mensaje entrante y la RLS está activa
- **THEN** el handler de webhook SHALL crear la conversación y el contacto sin error (INSERT permitido vía rol `app_session`)
- **AND** la conversación se crea con `assignee_stytch_member_id` según el auto-match (o NULL)

#### Scenario: Webhook no re-asigna

- **WHEN** el webhook procesa una actualización de estado de entrega
- **THEN** SHALL actualizar solo metadata de sistema
- **AND** SHALL NO modificar `assignee_stytch_member_id`

### Requirement: Workers de background fuera del scope de miembro (sin inanición)

Los paths de background que leen `crm.conversations`/`crm.messages` sin contexto de miembro (outbound send, message-send, campañas, analytics, pipeline durable, cron) SHALL ejecutarse con rol `app_session` (bypass RLS) o con contexto de organización explícito (`app.current_organization_id`), NUNCA con contexto de miembro inventado. Su control de aislamiento SHALL seguir siendo el query layer org-scoped existente (comportamiento actual).

#### Scenario: Worker de envío bajo RLS activa

- **WHEN** un worker de outbound/message-send lee conversaciones con RLS activa
- **THEN** la lectura SHALL ejecutarse con rol `app_session` (o contexto org explícito)
- **AND** SHALL NO devolver cero filas por ausencia de vars de miembro
- **AND** el query layer org-scoped SHALL seguir acotando al org del worker
