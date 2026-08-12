# inbox-ui Delta Spec

## ADDED Requirements

### Requirement: Picker de tres jugadas de ownership

El sistema SHALL proveer un componente de asignación (picker) disparado desde el chip de assignee en la cabecera del thread, que cubra tres jugadas: (1) **Reclamar** (conversación sin asignar → al miembro, 1 click); (2) **Transferir** (a otro miembro, con búsqueda); (3) **Liberar a la cola** (miembro → sin asignar, con confirmación y undo de 5s). El picker SHALL mostrar la lista de miembros de la organización (misma fuente que la gestión de miembros) y SHALL requerir `inbox:reassign` para transferir/liberar y para reclamar.

#### Scenario: Reclamar desde la cola

- **WHEN** un miembro con `inbox:view_unassigned` y `inbox:reassign` abre una conversación sin asignar y pulsa Reclamar
- **THEN** la conversación SHALL pasar a "Mis chats" con el miembro como assignee
- **AND** SHALL salir de la cola para el resto

#### Scenario: Transferir a un colega

- **WHEN** un miembro con `inbox:reassign` transfiere una conversación visible a otro miembro
- **THEN** SHALL actualizar el assignee server-side
- **AND** el receptor SHALL recibir la señal de llegada (toast + "nueva")

#### Scenario: Liberar a la cola con undo

- **WHEN** un miembro con `inbox:reassign` libera una conversación a la cola
- **THEN** SHALL pedir confirmación
- **AND** SHALL ofrecer "Deshacer" durante 5 segundos
- **AND** si el chat fue reclamado por otro antes del undo, SHALL mostrarse "Ya fue reclamada por X" sin estado inconsistente

#### Scenario: Sin permiso no hay picker

- **WHEN** un miembro sin `inbox:reassign` ve una conversación
- **THEN** el chip SHALL ser de solo lectura (sin picker)
- **AND** el backend SHALL rechazar la operación (403) si se intenta directo

### Requirement: Auto-claim al primer reply

Enviar un mensaje en una conversación sin asignar SHALL reclamarla implícitamente al remitente (si tiene permiso), de forma atómica con el envío. Si la conversación fue reclamada por otro entre la apertura y el envío, el mensaje SHALL enviarse igual y la UI SHALL indicar el dueño actual.

#### Scenario: Reply reclama

- **WHEN** un miembro envía un mensaje en una conversación con `assignee IS NULL`
- **THEN** la conversación SHALL quedar asignada a él en la misma transacción
- **AND** SHALL mostrarse el toast "Chat reclamado por ti" con acción "Devolver a la cola"

#### Scenario: Carrera de auto-claim

- **WHEN** dos miembros responden simultáneamente una conversación sin asignar
- **THEN** el primer envío reclama la conversación
- **AND** el segundo SHALL enviar su mensaje igual
- **AND** SHALL ver "Ahora asignada a X" en la cabecera

### Requirement: Banner de ownership en el poll

Si la conversación abierta cambia de assignee desde que se abrió (reclamada, transferida o liberada), el poll de 5s SHALL mostrar un banner en la cabecera ("Reasignada a Ana" / "Reclamada por Ana" / "Devuelta a la cola") sin bloquear el envío. El banner SHALL aparecer solo cuando el assignee cambió (diff), no en cada poll.

#### Scenario: Reasignación detectada en poll

- **WHEN** otra persona reasigna la conversación abierta
- **THEN** el siguiente poll SHALL mostrar el banner de ownership con el nuevo dueño

#### Scenario: Sin cambio no hay banner

- **WHEN** el assignee no cambió entre polls
- **THEN** SHALL NO mostrarse banner

### Requirement: Llegada de asignación al receptor

El receptor de una conversación asignada/transferida SHALL ver un toast in-app "Conversación asignada a ti" y la conversación SHALL aparecer en "Mis chats" con el punto "nueva" (distinto del no-leído).

#### Scenario: Toast de llegada

- **WHEN** un miembro recibe una conversación por transferencia o reasignación
- **THEN** SHALL mostrarse el toast in-app
- **AND** la conversación SHALL aparecer en "Mis chats" con punto "nueva"

### Requirement: Menú contextual de fila para reasignar

Las filas de la lista SHALL ofrecer un menú contextual ("⋯" → "Asignar a…") que abre el mismo picker, para reasignar sin abrir el thread. SHALL requerir `inbox:reassign` y visibilidad de la conversación.

#### Scenario: Reasignar desde la fila

- **WHEN** un miembro con `inbox:reassign` usa el menú contextual de una fila visible
- **THEN** SHALL abrir el picker con la conversación preseleccionada
- **AND** la transferencia SHALL aplicar el mismo flujo (auditoría incluida)
