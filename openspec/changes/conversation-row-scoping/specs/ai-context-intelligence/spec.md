# ai-context-intelligence Delta Spec

## ADDED Requirements

### Requirement: Contexto de conversación acotado por el scope del miembro

La generación de contexto IA para una conversación SHALL leer el historial de conversación a través del repositorio que aplica el predicado de scope del miembro. El sistema SHALL NO consultar `crm.conversations`/`crm.messages` mediante scans directos fuera del repositorio scoped (incluido el SQL de agentes en `agent.sql`). El contexto LLM SHALL quedar estrictamente acotado por la visibilidad del miembro solicitante.

#### Scenario: Miembro genera contexto de conversación visible

- **WHEN** un miembro solicita contexto de una conversación dentro de su scope
- **THEN** el historial SHALL obtenerse con el predicado de scope aplicado
- **AND** el contexto generado SHALL incluir solo mensajes de esa conversación

#### Scenario: Conversación fuera de scope no genera contexto

- **WHEN** un miembro solicita contexto de una conversación fuera de su scope
- **THEN** la consulta de historial SHALL devolver cero filas
- **AND** el endpoint SHALL devolver 404 o estado equivalente (sin filtrar existencia)

#### Scenario: Sin scans directos

- **WHEN** se audita el acceso a datos del módulo de contexto IA
- **THEN** SHALL NO existir consultas a `crm.conversations`/`crm.messages` fuera del repositorio con predicado de scope
