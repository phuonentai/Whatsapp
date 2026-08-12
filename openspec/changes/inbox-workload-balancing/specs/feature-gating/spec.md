# feature-gating Delta Spec

## ADDED Requirements

### Requirement: Workload solo en planes pagos

Los indicadores de capacidad, la configuración de límites y la vista de workload de equipo SHALL estar disponibles solo con el flag `conversation_row_scoping` activo. En free tier SHALL NO existir ninguna superficie de workload.

#### Scenario: Free tier sin workload

- **WHEN** el flag `conversation_row_scoping` está desactivado
- **THEN** SHALL NO mostrarse indicadores de capacidad ni configuración de límites
- **AND** la bandeja SHALL permanecer idéntica al estado previo

#### Scenario: Plan pago con workload

- **WHEN** el flag está activo
- **THEN** los indicadores de capacidad y la configuración de límites SHALL estar disponibles según permisos
