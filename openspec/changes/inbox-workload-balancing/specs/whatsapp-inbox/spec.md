# whatsapp-inbox Delta Spec

## ADDED Requirements

### Requirement: Límites de asignación configurables

El sistema SHALL soportar límites de conversaciones activas por miembro: defaults por rol (configurables por organización) con override individual por `stytch_member_id`. El conteo de carga SHALL considerar conversaciones con `assignee = miembro` y `status = 'active'`. La configuración SHALL requerir el permiso `inbox:manage_limits` (política Stytch, `admin`).

#### Scenario: Default por rol

- **WHEN** un miembro no tiene override individual
- **THEN** su límite SHALL ser el default de su rol

#### Scenario: Override individual

- **WHEN** un admin configura un límite individual para un miembro
- **THEN** SHALL reemplazar el default del rol para ese miembro

#### Scenario: Conteo de activas

- **WHEN** se calcula la carga de un miembro
- **THEN** SHALL contar conversaciones asignadas con `status = 'active'`
- **AND** las cerradas/archivadas SHALL NO contar

#### Scenario: Configuración sin permiso

- **WHEN** un miembro sin `inbox:manage_limits` intenta modificar límites
- **THEN** el backend SHALL rechazar (403)
- **AND** la UI SHALL ocultar los controles de configuración
