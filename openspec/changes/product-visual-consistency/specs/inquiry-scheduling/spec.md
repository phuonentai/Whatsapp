# inquiry-scheduling Delta Spec

## MODIFIED Requirements

### Requirement: Procurement con chips de estado semánticos

Las vistas de procurement y cronogramas de consulta SHALL presentar chips de estado (cotización, aprobación, programado, completado) con color + texto y el lenguaje visual del diseño. El flujo de negocio existente (crear consulta, agendar, responder) SHALL permanecer sin cambios.

#### Scenario: Chip de estado con texto

- **WHEN** un estado de proveedor o cronograma se muestra
- **THEN** el chip SHALL combinar color y texto legible
