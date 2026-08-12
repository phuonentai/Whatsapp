# whatsapp-webhook-ingress Delta Spec

## ADDED Requirements

### Requirement: Auto-match determinístico org-scoped de conversaciones entrantes

El path de ingestión SHALL intentar un match determinístico antes de dejar una conversación en la cola de no-asignados: resolver la empresa (`crm.companies`) por teléfono o NIT del contacto entrante **dentro del org resuelto del `phone_number_id` del payload** (`whatsapp.whatsapp_configs.phone_number_id` es UNIQUE → un solo org). Si hay match, auto-asignar `assignee_stytch_member_id = companies.owner_account_id` (vía `accounts.stytch_member_id`). Si no hay match (ni empresa ni historial de contacto en ese org), la conversación SHALL crearse con `assignee_stytch_member_id = NULL` (cola). El match SHALL NUNCA cruzar tenants: un teléfono/NIT puede existir en varias organizaciones de la plataforma y solo el org del `phone_number_id` es candidato.

#### Scenario: Match por empresa asigna al owner (mismo org)

- **WHEN** un inbound llega de un teléfono asociado a una `crm.companies` del mismo org que el `phone_number_id`, con `owner_account_id` definido
- **THEN** la conversación SHALL crearse con `assignee_stytch_member_id` = `stytch_member_id` del owner
- **AND** SHALL NO caer en la cola de no-asignados

#### Scenario: Match cross-tenant rechazado

- **WHEN** el teléfono/NIT matchea una empresa de OTRA organización distinta al org del `phone_number_id`
- **THEN** la conversación SHALL crearse sin auto-asignación (`assignee_stytch_member_id = NULL`, cola)
- **AND** SHALL NO asignarse al owner de la empresa de otro tenant

#### Scenario: Sin match cae a la cola

- **WHEN** un inbound no matchea ninguna empresa ni contacto previo en el org
- **THEN** la conversación SHALL crearse con `assignee_stytch_member_id = NULL`
- **AND** SHALL ser visible solo para miembros con `inbox:view_unassigned`

#### Scenario: El auto-match es determinístico (sin round-robin)

- **WHEN** múltiples inbounds llegan sin match
- **THEN** SHALL NO rotarse ni distribuirse por carga entre miembros
- **AND** todos SHALL quedar en la cola de no-asignados

### Requirement: Asignación inicial no re-asigna conversaciones existentes

El auto-match SHALL aplicarse solo en la creación de la conversación (inbound net-nuevo). Una conversación existente asignada SHALL conservar su `assignee_stytch_member_id` al recibir mensajes posteriores.

#### Scenario: Mensaje posterior no cambia asignación

- **WHEN** llega un inbound de un contacto con conversación ya asignada
- **THEN** la conversación SHALL conservar su assignee
- **AND** el auto-match SHALL NO sobreescribirlo
