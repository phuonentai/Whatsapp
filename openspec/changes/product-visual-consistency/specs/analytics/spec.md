# analytics Delta Spec

## MODIFIED Requirements

### Requirement: Reportes con paleta de tokens y estados honestos

Las vistas de reportes/analytics SHALL usar la paleta de tokens (`--chart-*`) para las series, leyendas con texto, y estados vacíos honestos (sin fabricar datos). Las tablas SHALL usar `th scope="col"` y chips de estado con texto. Los endpoints de analytics existentes y el gating por módulo (`analytics_module`) SHALL permanecer sin cambios.

#### Scenario: Series con tokens

- **WHEN** un chart de reportes renderiza series
- **THEN** SHALL usar la paleta de tokens del tema
- **AND** las leyendas SHALL incluir texto (no solo color)

#### Scenario: Estado vacío honesto

- **WHEN** no hay datos para el periodo consultado
- **THEN** SHALL mostrarse el estado vacío sin fabricar valores
