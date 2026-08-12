# knowledge-base-ui Delta Spec

## MODIFIED Requirements

### Requirement: Knowledge base con rail de dos modos

La página de knowledge (`app/dashboard/knowledge`) SHALL presentar un rail con modo segmentado **Chat / Documentos** (`?mode=chat|docs` en URL), un pane principal que alterna thread ⇄ detalle de documento, y un pane de contexto (xl+) que en chat muestra las fuentes de la última respuesta (título, icono por tipo, score de similitud, salto al documento) y en docs muestra la metadata del documento seleccionado. En viewports < 1280px el pane de contexto SHALL colapsar a drawer; en móvil SHALL ocultarse. El rail SHALL usar el mismo frame físico que la bandeja para coherencia del sistema.

#### Scenario: Modo Chat

- **WHEN** el usuario abre `/dashboard/knowledge?mode=chat`
- **THEN** SHALL renderizar el thread de chat con el rail de sesiones y el pane de contexto de fuentes

#### Scenario: Modo Documentos

- **WHEN** el usuario abre `/dashboard/knowledge?mode=docs`
- **THEN** SHALL renderizar la librería de documentos con el detalle del seleccionado y su metadata

#### Scenario: Estado de modo en URL

- **WHEN** el usuario cambia de modo
- **THEN** el parámetro `mode` de la URL SHALL actualizarse (deep-link/refresh/back)

### Requirement: Chat con citas inline y fuentes saltables

El chat SHALL renderizar el streaming existente (markdown sanitizado, live region) y SHALL añadir: marcadores de cita `[1]`, `[2]` inline donde el modelo referencia fuentes; tarjetas de cita bajo el mensaje (título + icono por tipo + score de similitud como etiqueta legible, nunca raw JSON); botón copiar; y clic en una cita SHALL saltar al chunk fuente en el detalle del documento. Si RAG no encuentra chunk relevante, el asistente SHALL responder "No encontré esto en tus documentos — ¿puedes agregarlo?" en lugar de responder desde el vacío (guard anti-hallucination).

#### Scenario: Respuesta con citas

- **WHEN** el modelo responde referenciando fuentes
- **THEN** SHALL renderizar marcadores inline y tarjetas de cita con salto al documento

#### Scenario: Sin fuentes relevantes

- **WHEN** el RAG no encuentra chunks relevantes para la pregunta
- **THEN** el asistente SHALL indicar honestamente que no lo encontró en los documentos

### Requirement: Vida del documento con estados y retry

El documento subido SHALL aparecer inmediatamente en la lista con chip de estado: procesando (amber), procesado (emerald), error (red + botón Retry). Retry SHALL re-disparar solo la re-embedding (nunca re-subida). El header del chat SHALL mostrar un chip sutil "N documentos en indexación". La eliminación SHALL cascadear a embeddings y citas (las citas existentes muestran el fallback "Documento no disponible").

#### Scenario: Upload con estados

- **WHEN** un admin sube un PDF
- **THEN** el documento SHALL aparecer con estado procesando
- **AND** al completar el procesamiento SHALL pasar a procesado

#### Scenario: Error con retry

- **WHEN** el procesamiento de un documento falla
- **THEN** SHALL mostrarse el estado error con acción Retry
- **AND** Retry SHALL re-embedding sin re-subir el archivo

### Requirement: Créditos y gate de plan en knowledge

El header SHALL mostrar el chip de créditos (usados/máximo) que pasa a amber ≥80%. Cuando `credits_remaining = 0`, el composer SHALL deshabilitarse antes del envío (input disabled + banner "Créditos agotados — ver plan"), porque el guard rechaza con 402 antes de abrir el stream. Los `tokensUsed` por mensaje del asistente SHALL mostrarse sutilmente (tooltip). Si el entitlement `ai_assistant` está apagado, la página SHALL mostrar un upgrade gate completo, no un chat roto.

#### Scenario: Composer deshabilitado sin créditos

- **WHEN** el org tiene `credits_remaining = 0`
- **THEN** el composer SHALL deshabilitarse con banner de créditos agotados
- **AND** la respuesta del servidor (402) SHALL NOT intentarse

#### Scenario: Chip de créditos con umbral

- **WHEN** el uso de créditos alcanza ≥80%
- **THEN** el chip SHALL mostrar el estado amber

### Requirement: Documento restringido sin fuga de título

Un miembro sin `org:manage` que abre un link a un documento `admin_only` SHALL ver un estado "Documento restringido" (403-style) SIN revelar el título del documento. En la librería, los docs `admin_only` SHALL llevar un badge de lock para admins.

#### Scenario: Link directo a documento restringido

- **WHEN** un miembro sin permiso abre el detalle de un documento `admin_only` por link directo
- **THEN** SHALL renderizar el estado restringido sin mostrar el título

### Requirement: Empty state guiado

El primer uso SHALL mostrar "Agrega tu primer documento" con un botón de PDF de muestra y una explicación de 3 líneas de lo que el asistente puede hacer; nunca una dropzone vacía muerta. El checklist de `ai-onboarding` SHALL enlazar aquí.

#### Scenario: Primer uso guiado

- **WHEN** el org no tiene documentos y el usuario abre el modo Documentos
- **THEN** SHALL mostrarse el estado vacío guiado con CTA de ejemplo
