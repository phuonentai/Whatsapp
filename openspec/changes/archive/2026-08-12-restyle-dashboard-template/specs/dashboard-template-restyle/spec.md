## ADDED Requirements

### Requirement: Dashboard shell con identidad del template

El panel del producto (`app/dashboard/*`) SHALL presentar la identidad visual del template de referencia: sidebar `slate-900` con grupos de navegación (Dashboard, Conversaciones, Contactos, Facturas, Pagos, Analíticas; grupo "Inteligencia Artificial": Copiloto IA, Entrenamiento, Automatizaciones), tarjeta "IA Insights", y top bar con búsqueda, notificaciones y usuario. La navegación SHALL mantener las rutas y permisos existentes; los tokens shadcn del tema SHALL conservarse para el contenido.

#### Scenario: Shell oscuro con navegación existente

- **WHEN** un usuario autenticado abre cualquier página de `/dashboard`
- **THEN** el shell SHALL renderizar la sidebar oscura con los ítems de navegación del template y las rutas reales de la app
- **AND** la navegación existente (bandeja, CRM, reportes, knowledge, settings) SHALL seguir funcionando con los permisos RBAC actuales

#### Scenario: Grupos del template respetan RBAC

- **WHEN** el grupo "Inteligencia Artificial" o la tarjeta "IA Insights" se renderizan en la sidebar
- **THEN** cada ítem SHALL mostrarse solo si el usuario tiene el permiso/entitlement correspondiente según el modelo de navegación filtrada existente
- **AND** los ítems sin ruta real o sin permiso SHALL NOT enlazarse incondicionalmente

### Requirement: Overview con KPIs reales y verificación de pagos preservada

`app/dashboard/page.tsx` SHALL mostrar un resumen con tarjetas KPI (conversaciones activas, ventas de la semana, facturas emitidas, tiempo de respuesta IA) alimentadas por las fuentes de datos existentes (inbox, analytics, invoicing, métricas de agente), reutilizando las queries que ya consume `DashboardHome` sin fan-out nuevo. Cuando una métrica no tenga fuente de datos, la tarjeta SHALL mostrar "—" y NUNCA una cifra inventada. La página SHALL conservar sin cambios la verificación de parámetros de pago previa al renderizado: `checkout_id` (Polar) vía `verifyPayment` y `payment_id`/`preapproval_id` (MercadoPago) vía `verifyMercadoPagoPayment`, con sus redirects a `/dashboard/settings?view=subscription&payment_{verified,error}=true`.

#### Scenario: Métricas con fuente de datos

- **WHEN** el módulo de datos correspondiente expone la métrica
- **THEN** la tarjeta SHALL mostrar el valor real vía las queries existentes

#### Scenario: Métrica sin fuente de datos

- **WHEN** no existe fuente de datos para una métrica
- **THEN** la tarjeta SHALL mostrar "—" sin fabricar valores

#### Scenario: Verificación de parámetros de pago preservada

- **WHEN** un usuario vuelve de un checkout con `checkout_id`, `payment_id` o `preapproval_id` en el query string
- **THEN** la página SHALL ejecutar la verificación correspondiente y redirigir a `/dashboard/settings?view=subscription` con `payment_verified=true` o `payment_error=true` según el resultado
- **AND** el restyle del overview SHALL NOT alterar estas ramas de verificación

### Requirement: Bandeja con composición messages-view

La bandeja (`app/dashboard/inbox`) SHALL presentar la composición del template `messages-view.html`: tarjetas de métricas (conversaciones hoy, por responder, tasa de respuesta, tiempo promedio), toolbar con búsqueda y filtros por canal y estado, y lista de conversaciones con avatar, etiqueta, snippet, hora y badge de no leídos — sobre la lógica de datos existente (inbox store/queries), sin cambios de comportamiento. El filtro por agente SHALL renderizarse solo si el modelo de datos expone el agente de la conversación; el modelo `Conversation` actual no lo expone, por lo que la bandeja filtra por canal (`whatsapp`/`instagram`) y estado.

#### Scenario: Bandeja operativa con datos reales

- **WHEN** un usuario abre la bandeja con conversaciones existentes
- **THEN** las métricas y la lista SHALL reflejar los datos reales del inbox
- **AND** los filtros de canal y estado SHALL filtrar la lista según la lógica existente

### Requirement: Onboarding con wizard del template

El flujo de signup (`app/signup`) SHALL presentar el wizard oscuro del template `onboarding.html` (progreso, navegación prev/next, selección de tipo de negocio) sobre la lógica existente (`use-signup-flow.ts`, componentes Stytch). Los contratos Stytch (magic link, redirects, RBAC) SHALL permanecer sin cambios.

#### Scenario: Flujo completo sin cambios de contrato

- **WHEN** un nuevo cliente completa el signup
- **THEN** el wizard SHALL pasar por los pasos account→organization→business con las validaciones actuales
- **AND** las llamadas a Stytch (`sendMagicLink`) y los redirects SHALL ser idénticos a los actuales
