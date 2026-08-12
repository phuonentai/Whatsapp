# billing-provider-ux Delta Spec

## MODIFIED Requirements

### Requirement: Subscription con uso visible y lenguaje del diseño

La vista de subscription SHALL presentar el lenguaje visual del diseño e integrar la visualización de uso (créditos IA y facturas incluidas/usadas con barras y umbral amber ≥80%). La lógica de negocio existente (estado Polar/MercadoPago, cancelación con confirmación, resumen, cambio de plan, `payment_verified`/`payment_error`) SHALL permanecer sin cambios.

#### Scenario: Plan y uso visibles

- **WHEN** el usuario abre la vista de subscription con un plan activo
- **THEN** SHALL mostrar el plan, el estado de renovación/cancelación y las barras de uso en el lenguaje del diseño
- **AND** el flujo de cancelación/resumen SHALL ser idéntico al previo

#### Scenario: Retorno de pago sin cambios

- **WHEN** un usuario vuelve de checkout con `payment_verified=true` o `payment_error=true`
- **THEN** la vista SHALL mostrar el banner correspondiente con la lógica existente intacta
