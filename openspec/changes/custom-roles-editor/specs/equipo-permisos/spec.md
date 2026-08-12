# equipo-permisos Delta Spec

## MODIFIED Requirements

### Requirement: Roles personalizados en matriz y asignación

Cuando existan roles personalizados (definidos vía `custom-roles-editor`), la matriz de permisos read-only SHALL incluirlos como columnas y el role select de asignación de miembros SHALL listarlos con su descripción. La asignación sigue vía member API.

#### Scenario: Rol personalizado en la matriz

- **WHEN** existe un rol personalizado activo
- **THEN** la matriz SHALL mostrarlo como columna con sus permisos efectivos
- **AND** el role select de miembros SHALL permitir asignarlo
