# dashboard-template-restyle Delta Spec

## MODIFIED Requirements

### Requirement: Overview con KPIs reales, recomposición de operación y verificación de pagos preservada

La recomposición del overview SHALL implementarse en el componente cliente `app/dashboard/components/dashboard-home.tsx` (`DashboardHome`), que SHALL presentar el resumen de operación según el export de diseño: saludo con fecha, selector de periodo y CTA "Nueva Conversación"; fila de 4 tarjetas KPI con chip de icono (conversaciones activas, ventas de la semana, facturas emitidas, tiempo de respuesta IA) y badge de delta cuando exista dato de comparación de periodo; chart "Rendimiento de Ventas" con selector de periodo y leyenda de ventas reales vs predicción IA; panel "Copiloto IA" con tarjetas de insight; paneles de **Conversaciones Recientes**, **Rendimiento del Equipo** y **Facturas Siigo**; banner **Auto-Piloto**; y **Acciones Rápidas** (Broadcast → campañas si existe ruta real, Nueva Factura → siigo, Nuevo Contacto → CRM, Exportar → reportes). Las KPIs, el chart y los paneles SHALL alimentarse de las fuentes de datos existentes (inbox, analytics, members, agent-settings) reutilizando las queries que ya consume `DashboardHome` sin fan-out nuevo ni endpoints nuevos. Cuando una métrica no tenga fuente de datos, la tarjeta o panel SHALL mostrar "—" o un estado vacío con CTA y NUNCA una cifra inventada. El server component `app/dashboard/page.tsx` SHALL conservar sin cambios su única responsabilidad de verificación de parámetros de pago previa al renderizado de `<DashboardHome />`: `checkout_id` (Polar) vía `verifyPayment` y `payment_id`/`preapproval_id` (MercadoPago) vía `verifyMercadoPagoPayment`, con sus redirects a `/dashboard/settings?view=subscription&payment_{verified,error}=true`; la recomposición NO SHALL alterar estas ramas de verificación.

#### Scenario: Métricas con fuente de datos

- **WHEN** el módulo de datos correspondiente expone la métrica
- **THEN** la tarjeta o panel SHALL mostrar el valor real vía las queries existentes
- **AND** el badge de delta SHALL renderizarse solo cuando exista dato de comparación de periodo

#### Scenario: Métrica sin fuente de datos

- **WHEN** no existe fuente de datos para una métrica o widget (facturas emitidas, tiempo respuesta IA, lista de facturas Siigo, rendimiento por miembro, predicción IA)
- **THEN** la tarjeta o panel SHALL mostrar "—" o un estado vacío con CTA sin fabricar valores

#### Scenario: Widgets heredan el gate de su superficie fuente

- **WHEN** un widget de la home se alimenta de una query de un módulo (inbox, analytics, members, siigo, agent-settings)
- **THEN** el widget SHALL renderizarse solo bajo las mismas condiciones de permiso/entitlement que su superficie fuente: el chart/KPI de ventas bajo `useModule("analytics").enabled` AND `hasPermission(invoice:view)`; el widget y KPI de Facturas Siigo bajo `hasPermission(invoice:view)`; el panel de Rendimiento del Equipo bajo el gate de la superficie de miembros que el modelo exponga; el banner Auto-Piloto bajo la misma condición de acceso a `settings?view=ai`
- **AND** ningún widget SHALL exponer datos a un miembro que no tenga el permiso de su módulo fuente (sin ensanchar acceso por renderizar en home); si no hay gate explícito, el widget SHALL mostrar estado vacío honesto

#### Scenario: Verificación de parámetros de pago preservada

- **WHEN** un usuario vuelve de un checkout con `checkout_id`, `payment_id` o `preapproval_id` en el query string
- **THEN** la página SHALL ejecutar la verificación correspondiente en `app/dashboard/page.tsx` y redirigir a `/dashboard/settings?view=subscription` con `payment_verified=true` o `payment_error=true` según el resultado
- **AND** la recomposición del overview en `dashboard-home.tsx` SHALL NOT alterar estas ramas de verificación

### Requirement: Helpers de onboarding en la home

Los helpers de primer uso (`AssistantIntro`, `FirstRunChecklist`) SHALL permanecer disponibles en la home, ya sea visibles por defecto o plegados en un patrón colapsable, sin cambiar los contratos de `ai-onboarding`/`feature-gating` (visibilidad por entitlement y estado de primer uso). El estado de completitud del checklist SHALL derivarse de las mismas fuentes existentes (config de WhatsApp, suscripción, conversaciones, marcadores `ai-onboarding` de primer uso) en cada render; la única preferencia nueva SHALL ser el plegado manual del colapsable, persistida en localStorage y sin estado de servidor.

#### Scenario: Checklist completo pliega el helper

- **WHEN** el checklist de primer uso está completo (según las fuentes existentes)
- **THEN** el helper SHALL poder plegarse o colapsarse sin romper su estado de completitud
- **AND** cuando el checklist no está completo, el helper SHALL seguir visible por defecto en primer uso

### Requirement: Minimización de datos personales en el panel de conversaciones

El panel de Conversaciones Recientes de la home SHALL renderizar únicamente datos a nivel de snippet que la bandeja ya expone (nombre/avatar del contacto, último mensaje truncado, hora relativa, badge de no leídos si el modelo lo expone) y SHALL NOT renderizar cuerpos completos de mensajes, hilos, ni ninguna superficie de exportación o transferencia de datos personales; la home SHALL NOT crear una superficie de datos mayor que la bandeja (Ley 1581/Habeas Data: minimización y retención según las specs `data-transfer`/`data-backup-recovery`).

#### Scenario: El panel muestra solo snippets

- **WHEN** la home renderiza conversaciones recientes
- **THEN** cada ítem SHALL mostrar snippet y hora relativa (mismos campos que la lista de la bandeja)
- **AND** el clic SHALL navegar a la bandeja (`/dashboard/inbox`), no abrir el hilo en home
