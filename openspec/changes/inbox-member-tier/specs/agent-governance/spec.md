# agent-governance Delta Spec

## MODIFIED Requirements

### Requirement: Envío manual de miembro con guardrails

El envío manual realizado por un miembro con `inbox:reply` SHALL atravesar los mismos guardrails de `send_message` que el envío de un agente/admin: kill switch, descuento máximo, términos prohibidos, términos de escalación (deny para autónomo), consentimiento, ventana de envío y límite diario. Las denials SHALL registrarse en el audit con el mismo formato.

#### Scenario: Miembro y guardrail deny

- **WHEN** un miembro envía manualmente un mensaje que viola un guardrail (p. ej. término prohibido)
- **THEN** el envío SHALL denegarse sin side effect
- **AND** la denial SHALL registrarse en el audit
