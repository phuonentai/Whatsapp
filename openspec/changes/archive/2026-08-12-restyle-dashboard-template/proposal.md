# Proposal: Restyle del panel NexoChat según el template de referencia (dashboard.html + messages-view.html + onboarding.html)

## Why

El cliente aprobó como referencia de diseño el template Shuffle (`next_b2b_starter/website/shuffle-20260811-1759-14517.zip`), que incluye además de la landing **tres composiciones de producto**: `dashboard.html` (panel de control con sidebar oscura, KPIs, gráfico de ventas y panel "Copiloto IA"), `messages-view.html` (bandeja de mensajes con tarjetas de métricas, toolbar y lista de conversaciones) y `onboarding.html`/`onboarding-info.html` (wizard oscuro multi-paso para nuevos clientes). Estas composiciones aplican al **panel real** (`app/dashboard/*`) — el producto actual usa el tema neutro shadcn (variables CSS 240 hue, primario negro) y no refleja la identidad emerald/slate del sitio nuevo, lo que rompe la coherencia marca↔producto.

El overview actual (`app/dashboard/page.tsx`) ya renderiza una vista de resumen (`DashboardHome` con tarjetas KPI, quick actions, checklist de primer uso e intro del asistente) precedida de lógica de verificación de parámetros de pago (Polar `checkout_id`, MercadoPago `payment_id`/`preapproval_id`); **no** es un simple redirect a settings. Este change alinea la presentación de esa vista con el template, conservando intacta la verificación de pagos y el resto de contratos de negocio: auth Stytch, RBAC, API y datos quedan intactos; es un cambio de presentación.

## What Changes

- **Dashboard shell** (`components/layout/*`): sidebar oscura `bg-slate-900` con grupos de navegación (Dashboard, Conversaciones, Contactos, Facturas, Pagos, Analíticas + grupo "Inteligencia Artificial": Copiloto IA, Entrenamiento, Automatizaciones) y tarjeta "IA Insights"; top bar con búsqueda (⌘K), notificaciones y usuario — según `dashboard.html`. Los grupos nuevos del template se renderizan a través del modelo existente de navegación filtrada por permisos/entitlement del sidebar (nada de enlaces incondicionales a secciones gated o inexistentes); la tarjeta "IA Insights" se muestra según permisos. Header/logo con la marca emerald de NexoChat (reutilizar `LogoMark` de `components/marketing/site-header.tsx`).
- **Dashboard overview real** (`app/dashboard/page.tsx`): recompone la vista existente sobre el template — saludo + selector de rango (si existe analytics), botón "Nueva Conversación", 4 tarjetas KPI (conversaciones activas, ventas semana, facturas emitidas, tiempo respuesta IA), gráfico de rendimiento de ventas y panel "Copiloto IA". **Se conservan sin cambios las ramas server-side de verificación de parámetros de pago** (`verifyPayment` para Polar `checkout_id`; `verifyMercadoPagoPayment` para MercadoPago `payment_id`/`preapproval_id`) y sus redirects a `/dashboard/settings?view=subscription&payment_{verified,error}=true`. Las KPIs reutilizan las queries existentes (las de `DashboardHome`: `useConversationsQuery` + queries CRM/analytics según existan); valores "—" cuando no haya fuente, sin fabricar cifras. El contenido existente (`DashboardHome` KPI cards, quick actions, checklist, intro del asistente) se reutiliza/restiliza — no se elimina.
- **Inbox restyle** (`app/dashboard/inbox/*` + `components/whatsapp/*` o `components/inbox/*`): aplicar la composición de `messages-view.html` — 4 tarjetas de métricas (conversaciones hoy, por responder, tasa de respuesta, tiempo promedio), toolbar con búsqueda + filtros por canal y estado (el filtro por agente solo si el modelo de datos lo expone; el modelo `Conversation` actual no lo hace), lista de conversaciones con avatar, etiqueta, snippet, hora y badge de no leídos — manteniendo la lógica de datos existente (inbox store, queries) intacta.
- **Onboarding/signup restyle** (`app/signup/*`, `hooks/use-signup-flow.ts` sin cambios de lógica): aplicar el wizard oscuro de `onboarding.html` (pasos con progreso, botones prev/next, tarjetas de tipo de producto) al flujo existente account→organization→business. Los contratos Stytch (magic link, redirects, RBAC) NO cambian: es estilización de los componentes existentes.
- **404 real**: `app/not-found.tsx` deja de redirigir y muestra un 404 con branding (ya entregado en el change de marketing; se mantiene y verifica).
- **Consistencia**: `app/globals.css`/`tailwind.config.ts` conservan los tokens existentes; el shell e inbox usan utilidades slate/emerald explícitas (identidad del template, shell fijo oscuro), el contenido de las páginas conserva los tokens de tema (bg-card, bg-primary) y next-themes sigue funcional.

## Capabilities

### New Capabilities

- `dashboard-template-restyle`: presentación del panel del producto alineada al template de referencia — shell con sidebar oscura, overview con KPIs reales y verificación de pagos preservada, bandeja tipo messages-view, wizard de onboarding oscuro; cero cambios de lógica de negocio

### Modified Capabilities

- `app-shell`: la requirement "Dashboard home shows KPIs and quick actions" se extiende (set de KPIs del template, regla "—") y la requirement de dark mode se ajusta para el shell fijo oscuro (delta spec incluido). `inbox-ui` y `signup-stytch-compliance` cambian solo en presentación; sus specs de comportamiento no se alteran.

## Impact

- **Frontend only**: `next_b2b_starter/components/layout/*`, `app/dashboard/page.tsx` (overview preservando verificación de pagos), `app/dashboard/inbox/*`, `app/signup/*`, `app/not-found.tsx` (ya hecho), copia en `lib/copy/ui.ts` (namespaces `dashboard`, `inbox`, `auth`, `onboarding` existentes; añadir claves nuevas con es/en). Sin backend, sin migraciones, sin SQLC.
- **Auth**: cero cambios de contrato Stytch (el wizard reutiliza `use-signup-flow.ts` y los componentes Stytch existentes; solo estilos). Rollback: git revert — sin estado Stytch que revertir.
- **Billing**: las ramas de verificación de pago de `app/dashboard/page.tsx` (Polar checkout return y MercadoPago preapproval/payment return) se conservan byte a byte en comportamiento; la verificación del flujo de retorno de checkout se añade explícitamente al gate.
- **Datos**: los KPIs del overview reutilizan las queries existentes (las de `DashboardHome`); ante datos ausentes se muestra "—" (nunca cifras inventadas). Sin nuevas queries fan-out.
- **Dependencias**: ninguna nueva (lucide-react, shadcn ya instalados; `LogoMark` reutilizado).
- **Ops**: `pnpm build`, `pnpm lint`, `pnpm dev` en `next_b2b_starter/`; verificación e2e Playwright de dashboard/inbox/signup existentes deben seguir pasando (sin cambios de selectores críticos de lógica — solo clases de estilo; si un test depende de clases, se actualiza el page-object). Verificación adicional: retorno de checkout Polar y MercadoPago sigue aterrizando en las vistas de suscripción correctas.
- **Non-Goals**: no nuevos features de dashboard; no migrar datos; no tocar el backend ni la API; no cambiar el flujo Stytch; no modificar la lógica de verificación de pagos; no tocar el sitio de marketing (change separado: `add-enterprise-marketing-website`); no rediseñar settings/CRM/reportes/knowledge (se benefician del shell nuevo pero su contenido no se recompone — follow-up).

## Assumptions

- El template `dashboard.html`, `messages-view.html` y `onboarding.html` son las composiciones aprobadas (extraídas en `website/RECON-CODE.md` + notas de diseño).
- `app/dashboard/page.tsx` verifica parámetros de pago (Polar `checkout_id`; MercadoPago `payment_id`/`preapproval_id`) y renderiza `DashboardHome` (KPIs + quick actions + checklist + intro del asistente). El restyle conserva esta estructura; la premisa verificada en el repo a fecha del change es que estas ramas existen y no deben tocarse.
- Los datos de los KPIs provienen de los módulos existentes (analytics, crm, whatsapp-inbox) vía las queries que ya consume `DashboardHome`; donde el repo no exponga el dato, se muestra "—" o el valor existente de la UI actual, sin fabricar cifras.
- El wizard de onboarding conserva los 3 pasos actuales y sus validaciones (`use-signup-flow.ts`); el template aporta la presentación.
- Los grupos de navegación del template (incluido "Inteligencia Artificial") se renderizan condicionalmente según permisos/entitlement existentes; secciones sin ruta real o sin permiso no se enlazan incondicionalmente.
