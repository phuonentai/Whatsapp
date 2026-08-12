# workspace-settings-management Delta Spec

## MODIFIED Requirements

### Requirement: Perfil y workspace con lenguaje del diseño

La sección de perfil/workspace SHALL presentar el lenguaje visual del diseño (superficies claras, tarjetas, jerarquía), manteniendo sin cambios los contratos existentes: edición del nombre del org vía `PUT /organizations` con campo `status`, sync a Stytch circuit-breaker-guarded, y RBAC (`org:manage` para editar; read-only sin permiso).

#### Scenario: Edición de nombre re-estilizada

- **WHEN** un admin edita y guarda el nombre del workspace
- **THEN** el formulario SHALL usar el lenguaje visual
- **AND** SHALL enviar `PUT /organizations` con `name` y `status` actuales exactamente como antes

#### Scenario: Sync Stytch sin cambios

- **WHEN** el breaker de Stytch está abierto durante `PUT /organizations` o `PUT /auth/members/:member_id/role`
- **THEN** SHALL retornar error sin escribir en local (contrato vigente intacto)
