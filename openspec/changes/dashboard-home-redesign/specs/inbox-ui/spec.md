# inbox-ui Delta Spec

## ADDED Requirements

### Requirement: Conversaciones recientes reutilizan la bandeja

La home del dashboard SHALL mostrar un panel de conversaciones recientes alimentado por `useConversationsQuery` (misma fuente que la bandeja), ordenado por `last_message_at` descendente y limitado a un subconjunto (p. ej. 4-5), con nombre/avatar del contacto, último mensaje, hora relativa y badge de no leídos cuando el modelo lo exponga. Cada ítem SHALL enlazar a la bandeja (`/dashboard/inbox`) o a la conversación si la ruta de la bandeja lo soporta. El panel NO SHALL duplicar la lógica de filtrado/estado de la bandeja.

#### Scenario: La home muestra conversaciones recientes

- **WHEN** el org tiene conversaciones activas
- **THEN** la home SHALL listar las más recientes con su snippet y hora
- **AND** el clic en una conversación SHALL navegar a la bandeja

#### Scenario: Sin conversaciones

- **WHEN** el org no tiene conversaciones
- **THEN** el panel SHALL mostrar un estado vacío con CTA a la bandeja

#### Scenario: El panel solo expone datos a nivel de snippet

- **WHEN** el panel de la home renderiza una conversación
- **THEN** SHALL mostrar únicamente los mismos campos de lista que expone la bandeja (nombre/avatar del contacto, último mensaje truncado, hora relativa, badge de no leídos si el modelo lo expone)
- **AND** SHALL NOT renderizar el cuerpo completo del hilo, ni ofrecer exportación o transferencia de mensajes desde la home
- **AND** el clic SHALL navegar a la bandeja, donde se conserva la superficie existente de lectura de hilo
