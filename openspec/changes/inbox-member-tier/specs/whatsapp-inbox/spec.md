# whatsapp-inbox Delta Spec

## MODIFIED Requirements

### Requirement: Bandeja con lectura y envío manual para miembros

El sistema SHALL permitir a miembros con `inbox:view`/`inbox:reply` (según la política Stytch) listar conversaciones del org, ver el thread y enviar respuestas manuales vía el path de envío existente. El envío manual SHALL atravesar la cadena existente (auth, org_context, guardrails de `agent-governance`, metering) sin excepción. Cerrar/reabrir conversaciones, quick replies y secuencias SHALL requerir `org:manage`.

#### Scenario: Miembro responde manualmente

- **WHEN** un miembro con `inbox:reply` envía una respuesta manual
- **THEN** el mensaje SHALL enviarse por el path existente con guardrails evaluados
- **AND** las denials de guardrail SHALL registrarse en el audit (contrato vigente)

#### Scenario: Miembro sin org:manage no cierra

- **WHEN** un miembro sin `org:manage` intenta cerrar una conversación
- **THEN** el sistema SHALL rechazar con 403
- **AND** la UI SHALL NO renderizar el control de cerrar

### Requirement: Lectura scoped al org

La lista y el thread SHALL permanecer scoped a la organización del miembro (tenancy por org vigente); el nuevo tier no cambia el aislamiento entre orgs.

#### Scenario: Miembro ve solo su org

- **WHEN** un miembro abre la bandeja
- **THEN** SHALL ver únicamente conversaciones de su org
