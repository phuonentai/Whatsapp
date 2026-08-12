# whatsapp-inbox Delta Spec

## ADDED Requirements

### Requirement: Auto-claim condicional en el envío de mensajes

El endpoint de envío de mensajes (POST mensajes) SHALL reclamar la conversación al remitente si está sin asignar y el remitente tiene permiso para reclamarla, en la misma transacción del envío (UPDATE condicional `WHERE assignee IS NULL`). Si la conversación ya fue reclamada por otro, el mensaje SHALL enviarse igual y la respuesta SHALL incluir el dueño actual. El comportamiento SHALL ser idempotente y no alterar los guardrails de envío existentes.

#### Scenario: Envío reclama en la misma transacción

- **WHEN** un miembro envía un mensaje en una conversación sin asignar
- **THEN** el assignee SHALL quedar en el remitente en la misma transacción que el mensaje
- **AND** no existe ventana donde el mensaje se envíe y la conversación siga sin asignar

#### Scenario: Carrera resuelta sin pérdida

- **WHEN** dos envíos concurrentes reclaman la misma conversación sin asignar
- **THEN** uno de ellos gana el claim (UPDATE condicional)
- **AND** ambos mensajes SHALL persistir
- **AND** la respuesta del perdedor SHALL indicar el dueño actual

#### Scenario: Guardrails intactos

- **WHEN** el auto-claim se ejecuta
- **THEN** los guardrails de envío (kill switch, consentimiento, ventana, límite diario) SHALL aplicarse igual
- **AND** los denials SHALL registrarse en audit (comportamiento existente)
