# inbox-ui Delta Spec

## ADDED Requirements

### Requirement: Selector de scope aditivo con contadores

La bandeja SHALL mostrar un selector de scope en forma de píldoras compactas con contadores ("Mis chats (n)", "Cola (n)", "Todos (n)") colocado por encima de las filas de tabs existentes de estado y canal. Las píldoras SHALL existir solo según permisos: "Mis chats" siempre; "Cola" solo con `inbox:view_unassigned`; "Todos" solo con `inbox:view_all` u `org:manage`. Las filas de tabs de estado (All/Active/Closed/Archived) y canal (All/WhatsApp/Instagram) SHALL permanecer intactas y funcionales.

#### Scenario: Miembro ve solo "Mis chats"

- **WHEN** un miembro con `inbox:view` abre la bandeja
- **THEN** SHALL ver la píldora "Mis chats" con el conteo de conversaciones en su scope
- **AND** las píldoras "Cola" y "Todos" SHALL NO mostrarse

#### Scenario: Cola y Todos según permisos

- **WHEN** un miembro con `inbox:view_unassigned` y `inbox:view_all` abre la bandeja
- **THEN** SHALL ver las tres píldoras con sus contadores
- **AND** cada píldora SHALL consultar el mismo predicado de scope del backend

#### Scenario: Tabs existentes intactos

- **WHEN** la bandeja carga con el selector de scope visible
- **THEN** los tabs de estado y canal existentes SHALL seguir funcionando dentro de la píldora seleccionada
- **AND** el layout master-detail SHALL no cambiar

### Requirement: Identidad de asignación en lista y cabecera

Cada fila de conversación SHALL mostrar el assignee como chip de avatar compacto (iniciales, color estable derivado del id). Una conversación sin asignar SHALL mostrar un slot vacío con anillo ámbar. La cabecera del thread SHALL mostrar el assignee actual con la misma gramática visual (mismo componente de chip).

#### Scenario: Fila asignada muestra chip

- **WHEN** una conversación tiene `assignee_stytch_member_id`
- **THEN** la fila SHALL mostrar el chip del assignee
- **AND** el mismo chip SHALL aparecer en la cabecera del thread

#### Scenario: Fila sin asignar muestra slot ámbar

- **WHEN** una conversación tiene `assignee_stytch_member_id IS NULL`
- **THEN** la fila SHALL mostrar un slot vacío con anillo ámbar
- **AND** SHALL NO ser confundible con una conversación asignada

### Requirement: Distinción "nueva" vs no-leído

Una conversación recién asignada al miembro (ownership llegó sin mensajes nuevos) SHALL mostrar un punto "nueva" distinto del badge de no-leído (mensajes nuevos). Ambos indicadores SHALL poder coexistir y renderizarse de forma diferenciada.

#### Scenario: Reasignación marca "nueva" sin no-leído

- **WHEN** una conversación es asignada al miembro sin mensajes nuevos desde la asignación
- **THEN** SHALL mostrar el punto "nueva"
- **AND** SHALL NO mostrar el badge de no-leído

#### Scenario: Mensajes nuevos después de asignación

- **WHEN** llegan mensajes nuevos a una conversación ya asignada al miembro
- **THEN** SHALL mostrar el badge de no-leído (comportamiento existente intacto)

### Requirement: Estados vacíos por scope

La bandeja SHALL distinguir tres estados vacíos: (1) sin conversaciones en el scope del miembro pero con cola → mensaje con CTA "Reclama de la cola (n)"; (2) cola vacía → mensaje de refuerzo; (3) sin permiso → el control/tab SHALL NO existir (nunca un listado vacío que sugiera falta de datos cuando es falta de permiso). El estado de loading (skeleton) y error (retry) existentes SHALL permanecer.

#### Scenario: Sin chats asignados con cola disponible

- **WHEN** un miembro tiene 0 conversaciones en "Mis chats" y la cola tiene 2
- **THEN** SHALL mostrar "No tienes chats asignados — reclama de la cola (2)" con CTA a la píldora Cola

#### Scenario: Cola vacía

- **WHEN** un miembro con `inbox:view_unassigned` abre la Cola y está vacía
- **THEN** SHALL mostrar "La cola está vacía — todo asignado"

#### Scenario: Sin permiso no hay evidencia

- **WHEN** un miembro sin `inbox:view_unassigned` ni `inbox:view_all` navega a la bandeja
- **THEN** SHALL NO existir píldoras de Cola/Todos ni ningún listado que sugiera conversaciones ocultas

### Requirement: Urgencia de cola con ventana 24h

Las conversaciones sin asignar SHALL mostrar una cuenta regresiva de la ventana comercial de WhatsApp (p. ej. "responder en 16h") con escala de color (ámbar → rojo) y la cola SHALL ordenarse por (urgencia, antigüedad). El badge de la píldora Cola SHALL ser discreto y SHALL pulsar sutilmente solo cuando existe urgencia (con anuncio live-region para a11y, sin sonido).

#### Scenario: Cola ordenada por urgencia

- **WHEN** la cola contiene conversaciones sin asignar con distinta ventana restante
- **THEN** SHALL ordenarse primero por urgencia (menor tiempo restante) y luego por antigüedad

#### Scenario: Countdown visible en fila

- **WHEN** una conversación sin asignar tiene menos de 24h desde el último inbound sin respuesta
- **THEN** la fila SHALL mostrar el tiempo restante con color según urgencia

### Requirement: Métricas por audiencia según scope

El panel de métricas de la bandeja SHALL bifurcar por audiencia: miembros con `inbox:view_all`/`org:manage` ven el strip org-wide; el resto ve mini-stats personales (conversaciones propias, tiempo de respuesta propio).

#### Scenario: Supervisor ve métricas org-wide

- **WHEN** un miembro con `inbox:view_all` abre la bandeja
- **THEN** SHALL ver el strip de métricas org-wide existente

#### Scenario: Miembro ve mini-stats personales

- **WHEN** un miembro sin `inbox:view_all` abre la bandeja
- **THEN** SHALL ver mini-stats personales de su scope
- **AND** SHALL NO ver métricas org-wide

### Requirement: Comportamiento free-tier y primer-run de upgrade

Con `conversation_row_scoping` desactivado (free tier), la bandeja SHALL ser idéntica al estado previo (sin píldoras de scope, sin chips de assignee, sin urgencia). Al activarse el flag: si existen conversaciones sin asignar, la bandeja SHALL abrir en la píldora Cola y SHALL mostrar un tooltip de una vez explicando las píldoras.

#### Scenario: Free tier intacto

- **WHEN** una organización no tiene el flag activo
- **THEN** la bandeja SHALL mostrar exactamente la UI previa (tabs existentes, sin capa de scope)

#### Scenario: Upgrade aterriza en la cola

- **WHEN** el flag se activa y la organización tiene conversaciones sin asignar
- **THEN** la bandeja SHALL abrir en la píldora Cola con el tooltip explicativo
