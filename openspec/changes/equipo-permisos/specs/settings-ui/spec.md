# settings-ui Delta Spec

## ADDED Requirements

### Requirement: Navegación de settings incorpora la vista de derechos

La navegación de settings (`settings-content.tsx`) SHALL incorporar la vista `?view=access` ("Equipo y permisos") con gate `org:manage`, registrándola en el allowlist de gates existente (mismo mecanismo que `?view=members`/`?view=subscription`), y SHALL enlazarla desde el overview. Las vistas `?view=members` y `?view=modules` SHALL permanecer disponibles (la consolidación no elimina rutas existentes ni duplica su estado).

#### Scenario: Vista access en el overview

- **WHEN** un admin abre el overview de settings
- **THEN** SHALL ver la sección "Equipo y permisos" con su resumen (nº de miembros, rol propio)
- **AND** el clic SHALL navegar a `?view=access`

#### Scenario: Gates de acceso por vista

- **WHEN** un usuario sin `org:manage` intenta abrir `?view=access`
- **THEN** SHALL recibir la vista 403 (sin datos) o el gate existente del stack

#### Scenario: Registro en el allowlist de gates

- **WHEN** el parámetro `?view=access` llega con permisos listos
- **THEN** la vista SHALL resolverse únicamente si `org:manage` está en el allowlist del stack de settings
- **AND** una vista no registrada SHALL caer al overview sin renderizar datos
