# Proposal: Dashboard home recomposition — steady-state ops + first-run checklist

## Why

El equipo de diseño UX/UI entregó un export Shuffle (`shuffle-20260812-1654-48171.zip`, página `dashboard.tsx`) que redefine la página de inicio del panel: además de las KPIs y el chart actuales, incorpora conversaciones recientes, rendimiento del equipo, facturas Siigo, banner Auto-Piloto y acciones rápidas orientadas a operación. La página actual (`DashboardHome`) tiene la identidad del template pero una composición más pobre (helpers de onboarding + deals por etapa + actividad reciente). Esta propuesta recompone la home para que sea la superficie de operación diaria que el diseño especifica, manteniendo la regla de honestidad de datos ("—" sin fuente) y la verificación de pagos intacta.

## What Changes

- **Recomposición de `app/dashboard/components/dashboard-home.tsx`** según `DashboardSectionCustomComponents2` del export: saludo + fecha + selector de periodo + CTA "Nueva Conversación"; fila de 4 KPIs con chip de icono **y badge de delta** (`+23%`) cuando exista dato de comparación de periodo; chart "Rendimiento de Ventas" con leyenda ventas reales vs predicción IA; panel "Copiloto IA" oscuro con tarjetas de insight; fila de 3 paneles: **Conversaciones Recientes**, **Rendimiento del Equipo**, **Facturas Siigo**; banner **Auto-Piloto** (CTA → `settings?view=ai`); **Acciones Rápidas** (Broadcast → campañas, Nueva Factura → siigo, Nuevo Contacto → CRM, Exportar → reportes).
- **Widgets con datos reales** (queries existentes, sin endpoints nuevos): conversaciones activas + recientes (unread via `metadata` cuando exista), ventas de la semana + chart (analytics revenue), rendimiento del equipo (miembros `useMembersQuery` + conteo de conversaciones/actividades por miembro si el modelo lo permite), Auto-Piloto (estado `mode` de `agent-settings`).
- **Widgets sin fuente → "—" o estado vacío, nunca cifras inventadas**: facturas emitidas (KPI), tiempo de respuesta IA (KPI), lista de facturas Siigo (muestra estado vacío + CTA), delta badges y predicción IA del chart (se omiten o se muestran como "—" cuando no haya serie de comparación). **Sin endpoints nuevos de backend en este change.**
- **Onboarding helpers** (`AssistantIntro`, `FirstRunChecklist`): se conservan en la home (el diseño no los incluye, pero son contratos de `ai-onboarding`/`feature-gating` ya firmados); el checklist ya se auto-oculta cuando está completo (verificado en código: `return null`), y la novedad es un patrón colapsable con plegado manual en estado parcial — decisión en design.md; los contratos de spec de onboarding no cambian, la única preferencia nueva es de plegado en localStorage.
- **Chrome sin cambios**: sidebar/top bar del shell y navegación RBAC intactos; la home solo cambia su contenido.
- **Verificación de pagos preservada**: `app/dashboard/page.tsx` (server component) NO se toca — `checkout_id` Polar y `payment_id`/`preapproval_id` MercadoPago con sus redirects quedan byte a byte.

## Capabilities

### New Capabilities

- (ninguna — la home se gobierna por deltas sobre capacidades existentes)

### Modified Capabilities

- `dashboard-template-restyle`: la requirement "Overview con KPIs reales y verificación de pagos preservada" cambia — la home se recompone al layout del export (KPIs con delta, chart con leyenda real vs predicción, panel Copiloto con tarjetas, conversaciones recientes, rendimiento del equipo, facturas Siigo con estado vacío, banner Auto-Piloto, acciones rápidas) manteniendo la regla de honestidad de datos y la verificación de pagos intacta.
- `inbox-ui`: las conversaciones recientes de la home SHALL enlazar a la bandeja existente y reutilizar `useConversationsQuery` (sin duplicar lógica de lista).

## Impact

- **Frontend only**: `next_b2b_starter/app/dashboard/components/dashboard-home.tsx` (recomposición) + `lib/copy/ui.ts` (copia española de los nuevos widgets). Sin backend, sin migraciones, sin SQLC, sin cambios de API (chart y KPIs reutilizan `analyticsRepository.revenue`, `useConversationsQuery`, `useMembersQuery`, `useAgentSettingsQuery` ya existentes).
- **Auth**: cero cambios Stytch (la home es contenido autenticado; RBAC de navegación intacto).
- **Billing**: ramas de verificación de pago de `app/dashboard/page.tsx` preservadas byte a byte.
- **Dependencias**: ninguna nueva (recharts ya presente; `useMembersQuery` ya existe).
- **Ops**: `pnpm build`, `pnpm lint`, `npx tsc --noEmit`; unit test existente `app/dashboard/components/dashboard-home.test.tsx` debe pasar (o actualizarse solo en clases/estructura); verificación de gates RBAC por widget y snippets sin cuerpos completos; visual/a11y Playwright 390x844/768x1024/1440x900 → `qa/`.
- **Rollback**: git revert del change; sin estado de DB ni de Stytch que revertir.
- **Non-Goals**: sin endpoints nuevos (facturas lista, tiempo respuesta IA, delta badges reales, predicción IA real quedan como change futuro de backend si el negocio los requiere); sin cambios de rutas; sin reescribir copy existente; sin tocar shell/RBAC/billing; sin almacenar credenciales localmente (todo auth sigue en Stytch B2B).

## Assumptions

- Los widgets del diseño con cifras hardcodeadas (1,847 / $47.2M / 284 / 1.2s / +23%…) son mockups; la implementación usa datos reales o "—", según la regla vigente de honestidad de datos.
- "Rendimiento del equipo" puede no tener métrica por miembro en el backend; en ese caso se muestra estado vacío con CTA a settings (no cifras inventadas).
- `agent-settings` expone el modo del asistente (`copilot`/`autopilot`) para el banner Auto-Piloto; si el modelo no lo expone, el banner se muestra como sugerencia estática (CTA a configuración) sin afirmar estado.
- Los helpers de onboarding se conservan (contratos existentes); su plegado por defecto se decide en design.md sin cambiar las specs de onboarding.
- **Verificado en código**: no existe hoy una ruta `/dashboard/campaigns` ni vista `view=campaigns` en settings para la acción Broadcast; si no existe ruta real al implementar, la acción se omite (regla D2.2 del shell).
- **Verificado en código**: el checklist de primer uso deriva su completitud de queries existentes y se auto-oculta al completarse; el estado colapsable nuevo es solo una preferencia de plegado en localStorage.
- **Económico**: este change no añade acciones de IA ni costos de LLM, ni cambios de precio/plan/créditos; la verificación de pagos queda byte a byte (verificado en `app/dashboard/page.tsx`).
