# inbox-ui Delta Spec

## MODIFIED Requirements

### Requirement: AI rail con distinción visual de contenido IA

La bandeja SHALL distinguir visualmente contenido generado por IA de contenido humano: los drafts y sugerencias de IA SHALL llevar el marcador ✦ con tinte violet suave; los mensajes humanos NO. Las sugerencias de escalación (`type=escalation`) SHALL presentarse en amber como "requiere juicio humano" y SHALL NEVER auto-enviarse. Aprobar una sugerencia SHALL precargar el composer para revisión humana (o exigir confirmación explícita de envío en dos pasos), nunca enviar en silencio.

#### Scenario: Draft IA distinguible

- **WHEN** el asistente genera un draft en el rail
- **THEN** SHALL mostrarse con ✦ y tinte violet
- **AND** el mensaje humano del mismo thread NO SHALL llevar la marca

#### Scenario: Escalación human-only

- **WHEN** existe una sugerencia de tipo escalación
- **THEN** SHALL mostrarse en amber con la nota "requiere juicio humano"
- **AND** SHALL NOT tener acción de auto-envío

#### Scenario: Aprobar precarga el composer

- **WHEN** el usuario aprueba una sugerencia de draft
- **THEN** el texto SHALL precargarse en el composer sin enviar
- **AND** el envío SHALL requerir la acción explícita del usuario

### Requirement: Composer con créditos agotados (402) solo para IA

Cuando los créditos se agotan, el menú de transformación IA del composer SHALL deshabilitarse con tooltip "Créditos agotados — ver plan" y el panel de sugerencias SHALL mostrar un aviso de una línea; el composer manual SHALL permanecer totalmente funcional. El 402 SHALL evaluarse antes de la llamada LLM (guard vigente).

#### Scenario: Sin créditos, el envío manual sigue

- **WHEN** el org tiene créditos = 0
- **THEN** las acciones IA SHALL deshabilitarse con tooltip
- **AND** el composer manual SHALL seguir enviando normalmente

### Requirement: Contexto consent-gated con degradación visible

Cuando un contacto retiró el consentimiento (Ley 1581), el tab Contexto SHALL mostrar una tarjeta estructural ("mensajes: N · consentimiento retirado") y SHALL NEVER renderizar hechos pre-generados; la degradación SHALL ser visible (nunca oculta) para que el operador sepa por qué el contexto es fino.

#### Scenario: Consentimiento retirado

- **WHEN** el contacto retiró el consentimiento y el usuario abre Contexto
- **THEN** SHALL mostrarse la tarjeta estructural sin hechos
- **AND** SHALL indicarse explícitamente el estado de consentimiento

### Requirement: Unread con live-region y banner de canal desconectado

La lista SHALL marcar no-leídos con dot + peso; al llegar nuevos mensajes en el poll de 5s, SHALL anunciarse una live-region ("2 nuevos") sin spam por poll. Cuando un canal está desconectado, SHALL mostrarse un banner sobre el composer ("WhatsApp desconectado — los envíos fallarán").

#### Scenario: Nuevos mensajes anunciados

- **WHEN** el poll detecta mensajes nuevos
- **THEN** SHALL actualizarse la lista (dot + bold)
- **AND** SHALL emitirse un anuncio live-region discreto

#### Scenario: Canal desconectado

- **WHEN** el canal seleccionado está desconectado
- **THEN** SHALL mostrarse el banner de desconexión sobre el composer

### Requirement: Secuencias con progreso

El modo secuencia SHALL mostrar un chip "Paso k de n" sobre el input; los pasos SHALL avanzar solo tras envío exitoso; cambiar de conversación SHALL resetear la secuencia; SHALL NEVER auto-enviarse.

#### Scenario: Progreso de secuencia

- **WHEN** el operador ejecuta una secuencia de guiones
- **THEN** SHALL mostrarse el chip de progreso
- **AND** el avance SHALL depender de envíos exitosos
