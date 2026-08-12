# inbox-ui Delta Spec

## ADDED Requirements

### Requirement: Indicadores de capacidad de asignación

El sistema SHALL mostrar la carga de asignación de un miembro como progreso ("6/8 conversaciones") en el picker de asignación (junto al miembro destino) y en las filas de conversaciones asignadas de la vista "Todos". Cuando el destinatario está al límite, el picker SHALL mostrar el estado en ámbar y SHALL requerir confirmación explícita para transferir igual.

#### Scenario: Progreso visible en el picker

- **WHEN** un miembro con `inbox:reassign` abre el picker para transferir
- **THEN** SHALL ver el progreso de carga de cada miembro listado

#### Scenario: Destinatario al límite

- **WHEN** el destinatario seleccionado tiene carga igual o mayor al límite
- **THEN** SHALL mostrarse "Ana está al límite (8/8)"
- **AND** la transferencia SHALL requerir confirmación explícita

#### Scenario: Límite no bloquea respuesta urgente

- **WHEN** un miembro al límite responde/auto-reclama una conversación urgente de la cola
- **THEN** el envío SHALL proceder
- **AND** el exceso de carga SHALL registrarse en audit

### Requirement: Vista de workload de equipo

La vista "Todos" SHALL ofrecer una agrupación por miembro con conteo y progreso de carga, visible solo para miembros con `inbox:view_all` u `org:manage`. La vista SHALL ser read-only.

#### Scenario: Supervisor ve carga del equipo

- **WHEN** un miembro con `inbox:view_all` usa la vista "Todos"
- **THEN** SHALL ver el conteo y progreso de carga por miembro
- **AND** SHALL NO poder modificar límites desde esta vista (read-only)

#### Scenario: Miembro sin view_all no ve cargas ajenas

- **WHEN** un miembro sin `inbox:view_all` navega la bandeja
- **THEN** SHALL NO ver la carga de otros miembros (solo la propia)
