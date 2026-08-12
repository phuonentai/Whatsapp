# signup-stytch-compliance Delta Spec

## ADDED Requirements

### Requirement: Signup wizard renders Verifika composition with visual DIAN steps

The signup flow (`app/signup`) SHALL present the Verifika wizard composition: welcome screen ("Bienvenido a NexoChat", "Configura tu empresa en 3 minutos"), company-info step (nombre de la empresa, NIT, tipo de régimen — Simplificado/Común —, ciudad de operación) annotated "Necesitamos estos datos para la facturación DIAN", and the "Conecta WhatsApp" step (migrar número actual / nuevo número, prefijo +57) — over the existing logic (`use-signup-flow.ts`, Stytch components). The DIAN fiscal fields SHALL be visual-only: the signup payload SHALL NOT include new fiscal fields, no local persistence SHALL be added, and all Stytch contracts in this capability (native invite flow with `SendInvite: true`, no `owner_password`, structured error codes) SHALL remain unchanged.

#### Scenario: Wizard shows DIAN steps without payload change

- **WHEN** a new client completes the signup wizard
- **THEN** the wizard SHALL display the company-info (NIT, régimen, ciudad) and WhatsApp steps in the Verifika composition
- **AND** the signup request SHALL NOT contain fiscal fields, `owner_password`, or any new field beyond the current DTOs

#### Scenario: Stytch contracts unchanged

- **WHEN** the wizard completes
- **THEN** the Stytch calls (`Members.Create` with `SendInvite: true`, magic link, redirects) SHALL be identical to the current flow
- **AND** structured error codes (STYTCH_UNAUTHORIZED, INVITE_FAILED, etc.) SHALL respond exactly as specified
