## MODIFIED Requirements

### Requirement: Billing surfaces render provider-appropriate copy

The system SHALL render provider-appropriate copy on billing surfaces (plans modal, subscription paywall, subscription tab) using the typed copy layer in Spanish-first voice, replacing the current mixed English/Spanish hardcoded strings. Payment method explanations (Polar international card, MercadoPago PSE / Nequi / Colombian card) SHALL be expressed in plain Spanish.

#### Scenario: Plans modal renders Spanish copy

- **WHEN** a user opens the plans modal
- **THEN** the modal heading, payment-method explanation, plan descriptions, and action labels SHALL be Spanish strings resolved from the copy layer

#### Scenario: Active-subscription notice renders Spanish

- **WHEN** the plans modal shows the active-subscription blocking notice
- **THEN** the notice SHALL be the Spanish string ("Suscripción activa" with an instruction to cancel the current subscription before switching)
