# feature-gating Delta Spec

## ADDED Requirements

### Requirement: La UI de scope se oculta con el flag de plan

La capa de scope de la bandeja (píldoras de scope, chips de assignee, urgencia de cola, estados vacíos por scope, split de métricas) SHALL renderizarse solo cuando `conversation_row_scoping` está activo. En free tier la bandeja SHALL ser pixel-identical al estado previo.

#### Scenario: Flag off sin capa de scope

- **WHEN** `FeatureService.IsEnabled(orgID, "conversation_row_scoping")` devuelve false
- **THEN** SHALL NO renderizarse ninguna píldora de scope, chip de assignee ni urgencia
- **AND** la lista SHALL mostrar todas las conversaciones (comportamiento previo)

#### Scenario: Flag on con capa de scope

- **WHEN** el flag devuelve true
- **THEN** la capa de scope SHALL renderizarse según permisos del miembro
