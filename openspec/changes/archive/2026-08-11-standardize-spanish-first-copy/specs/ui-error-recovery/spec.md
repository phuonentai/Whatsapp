## MODIFIED Requirements

### Requirement: Data-loading views render error and retry states

The system SHALL render error, empty, and retry copy using the typed copy layer in Spanish-first voice across the onboarding, billing, WhatsApp-config, dashboard, agent-settings, and inbox surfaces.

#### Scenario: Error states render Spanish copy

- **WHEN** a data-loading or mutation failure view renders an error title and description
- **THEN** the title and description SHALL be Spanish strings resolved from the copy layer
- **AND** the retry action label SHALL be the Spanish "Reintentar" (or equivalent) from the copy layer
