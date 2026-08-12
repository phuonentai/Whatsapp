# signup-stytch-compliance Delta Spec

## MODIFIED Requirements

### Requirement: Signup con lenguaje visual sin cambios de contrato

El wizard de signup (`app/signup/*`) SHALL presentar el lenguaje visual del diseño (mismas superficies/inputs que el resto del producto, pasos fiscales DIAN visuales ya presentes). `use-signup-flow.ts`, los componentes Stytch (magic link, passkeys, MFA), los redirects y las validaciones SHALL permanecer sin cambios.

#### Scenario: Wizard re-estilizado

- **WHEN** un nuevo cliente completa el signup
- **THEN** el wizard SHALL usar el lenguaje visual
- **AND** las llamadas a Stytch y los redirects SHALL ser idénticos a los actuales
