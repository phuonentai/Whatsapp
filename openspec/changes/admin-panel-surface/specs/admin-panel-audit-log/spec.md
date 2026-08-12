# admin-panel-audit-log Delta Spec

## MODIFIED Requirements

### Requirement: Auditoría cross-org en la plataforma

Además de la vista por-org en settings (`?view=audit`), el sistema SHALL exponer una vista de auditoría de plataforma que consulte eventos del ledger `ai_usage_events` y la actividad operativa de plataforma (estado de conexiones, suscripción), con filtros (org, tipo, fecha) y gate `platform:operate`. La vista por-org existente SHALL permanecer sin cambios. La vista cross-org SHALL estar acotada a datos operativos: SHALL NOT exponer actividad CRM (notas, llamadas, correos, reuniones, tareas, mensajes de WhatsApp), contactos ni contenido de conversaciones en v1 (purpose limitation Ley 1581); esos datos siguen disponibles solo en la vista por-org dentro de settings.

#### Scenario: Vista cross-org

- **WHEN** un operador con `platform:operate` abre Auditoría en el shell
- **THEN** SHALL listar eventos del ledger `ai_usage_events` y eventos operativos de múltiples orgs con filtros aplicables
- **AND** la vista por-org de settings SHALL seguir funcionando

#### Scenario: Sin datos de clientes en la vista cross-org

- **WHEN** un operador filtra la auditoría cross-org
- **THEN** SHALL NOT aparecer actividad CRM ni contenido de conversaciones de ninguna org
- **AND** SHALL listarse solo eventos del ledger de uso y eventos operativos (estado, suscripción, conexiones)

#### Scenario: Lecturas registradas

- **WHEN** un operador consulta la auditoría cross-org
- **THEN** la lectura SHALL quedar registrada en `platform_access_log` (actor, org objetivo, acción, timestamp)
