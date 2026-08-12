# crm-frontend Delta Spec

## MODIFIED Requirements

### Requirement: CRM con lenguaje visual semántico

Las superficies CRM (listas, tablas, tarjetas, badges de etapa/etiqueta, diálogos) SHALL presentar el lenguaje visual semántico: estados con color + texto (emerald = activo/concedido, amber = pendiente, red = error/destructivo, gray = neutro), acciones destructivas (eliminar contactos/empresas/negocios) en red consistente, botones primarios emerald, y nunca color-only. Las queries, mutaciones y validaciones existentes SHALL permanecer sin cambios.

#### Scenario: Badge de estado con texto

- **WHEN** una etapa de negocio o etiqueta se muestra en una lista CRM
- **THEN** el badge SHALL combinar color y texto legible

#### Scenario: Acción destructiva consistente

- **WHEN** un usuario confirma la eliminación de un contacto/empresa/negocio
- **THEN** el diálogo y el botón destructivo SHALL usar la semántica red del lenguaje
- **AND** el flujo de confirmación existente SHALL permanecer sin cambios
