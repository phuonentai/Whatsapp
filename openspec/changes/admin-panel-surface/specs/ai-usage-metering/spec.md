# ai-usage-metering Delta Spec

## MODIFIED Requirements

### Requirement: Lectura de plataforma del ledger de uso

El sistema SHALL exponer una superficie read-only de plataforma que agregue el ledger `ai_usage` por org y periodo (tokens, créditos, límite y % de uso) y presente tablas de tasa de modelo de referencia. El modelo de datos vigente (`ai_usage`, `ai_usage_events`, `quota_tracking.ai_credits_max`) SHALL permanecer sin cambios; no se añaden escrituras nuevas desde la plataforma. Las agregaciones platform-wide (todas las orgs de un periodo) SHALL validar en spike la forma de query y la cobertura de índices (los índices vigentes son org-first: `idx_ai_usage_org_period`, `idx_ai_usage_events_org_created`); SHALL imponer límites de paginación obligatorios server-side; si se requiere un índice period-first, SHALL introducirse como migración aditiva separada (expand-contract), nunca dentro de la superficie read-only.

#### Scenario: Agregado por org/periodo

- **WHEN** un operador consulta el uso de un periodo
- **THEN** SHALL agregarse por org los tokens y créditos del ledger
- **AND** el % de uso SHALL calcularse contra `ai_credits_max` sin mutar el ledger

#### Scenario: Agregado platform-wide con paginación

- **WHEN** un operador consulta todas las orgs de un periodo
- **THEN** SHALL aplicarse límites de paginación server-side
- **AND** la query SHALL estar validada en el spike contra la cobertura de índices (period-first)
- **AND** SHALL NOT añadirse índices dentro de este cambio si el spike lo requiere (expand-contract separado)
