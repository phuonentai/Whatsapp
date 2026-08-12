# Design: Dashboard home recomposition — steady-state ops + first-run checklist

## Context

- Export de diseño Shuffle (`shuffle-20260812-1654-48171.zip`, `src/pages/dashboard.tsx` + `DashboardSectionCustomComponents2`) define la home: header con saludo/fecha/periodo/CTA, 4 KPIs con delta, chart con leyenda real vs predicción, panel Copiloto oscuro, conversaciones recientes, rendimiento del equipo, facturas Siigo, banner Auto-Piloto, acciones rápidas.
- Estado actual: `app/dashboard/components/dashboard-home.tsx` (client) con saludo, periodo tabs, 4 KPIs (2 con "—"), chart de ventas (serie única), panel Copiloto (3 insights de copy), `AssistantIntro` + `FirstRunChecklist`, deals por etapa + actividad reciente, 4 acciones rápidas (bandeja/CRM/knowledge/settings).
- `app/dashboard/page.tsx` (server) hace verificación de pagos antes de renderizar `<DashboardHome />` — no se toca. **Verificado en código**: ramas `checkout_id` (Polar vía `verifyPayment`), `payment_id`/`preapproval_id` (MercadoPago vía `verifyMercadoPagoPayment`) y preapproval-only, con sus redirects a `/dashboard/settings?view=subscription&payment_{verified,error}=true`.
- Fuentes existentes (verificadas): `useConversationsQuery`, `useContactsQuery`, `useDealsQuery`, `usePipelinesQuery`, `useActivitiesQuery`, `analyticsRepository.revenue`, `useMembersQuery`, `useAgentSettingsQuery`, `useModule("analytics")`, `hasPermission(PERMISSIONS.INVOICE_VIEW)`, `PERMISSIONS.ORG_MANAGE` (gate del inbox), `PERMISSIONS.INVOICE_VIEW` (`invoice:view`).
- **Verificado**: `FirstRunChecklist` deriva su estado de completitud de queries existentes (`useWhatsAppConfigQuery`, `useSubscriptionQuery`, `useConversationsQuery`, marcadores `isAssistantIntroDismissed`/`isInboxVisited`) y **ya retorna `null` cuando está completo** — es decir, ya se oculta solo al completarse. Los marcadores locales (dismiss/visited) son los contratos existentes de `ai-onboarding`; el nuevo estado colapsable solo añade una preferencia de plegado en localStorage.
- **Verificado**: no existe hoy una ruta `/dashboard/campaigns` ni vista `view=campaigns` en settings; `use-campaign-queries.ts` existe como fuente de datos, pero la ruta de campañas debe confirmarse en implementación; si no hay ruta real, la acción Broadcast se omite (regla D2.2 del shell).
- Sin `routing.json` previo en el change dir; se agrega en esta revisión (advisory: council + playwright + iso).

## Goals / Non-Goals

**Goals:**
- Recomponer la home al layout del diseño con datos reales o estados honestos ("—"/vacío con CTA).
- Nuevos widgets: conversaciones recientes, rendimiento del equipo, facturas Siigo (vacío honesto), banner Auto-Piloto, acciones rápidas operativas, delta badges cuando haya comparación.
- Mantener verificación de pagos, queries existentes sin fan-out nuevo, y **los gates de permiso/entitlement de cada superficie fuente** (sin ensanchar acceso al renderizar en home).
- Preservar los contratos de onboarding (`ai-onboarding`/`feature-gating`) sin romper el primer uso.

**Non-Goals:**
- NO nuevos endpoints de backend (lista de facturas, tiempo respuesta IA, predicción IA, deltas reales, métrica por miembro quedan como change futuro).
- NO cambiar rutas, shell, RBAC, billing, copy existente de otros módulos.
- NO fabricar cifras: todo dato sin fuente → "—" / estado vacío.
- NO exponer en home datos que la superficie fuente no expone ya (snippets únicamente, sin cuerpos completos de mensajes, sin exportación/transferencia nueva de datos personales).
- NO cambios de precio, plan, créditos, metering de IA ni costo de LLM: este change no añade acciones de IA ni endpoints.

## Decisions

1. **Layout por filas del diseño** — se replica la composición del export (`DashboardSectionCustomComponents2`) adaptada al grid shadcn existente (1/2/4 cols responsive, `xl:grid-cols-12` para la fila de 3 paneles y la fila chart+panel). Alternativa (mantener el layout actual con widgets añadidos) se descarta: el valor del diseño es la composición operativa, no solo los widgets.
2. **Honestidad de datos por widget** — tabla de decisión:
   | Widget | Fuente | Sin fuente |
   |---|---|---|
   | Conversaciones activas/recientes | `useConversationsQuery` | "—" / vacío + CTA bandeja |
   | Ventas semana + chart | `analyticsRepository.revenue` (gate `analytics_module` + `invoice:view`) | "—" / vacío + CTA reportes |
   | Rendimiento equipo | `useMembersQuery` + conteo por miembro si el modelo lo permite | vacío + CTA settings |
   | Facturas emitidas / lista Siigo | sin endpoint de lista hoy | "—" / vacío + CTA settings siigo |
   | Tiempo respuesta IA | sin métrica | "—" |
   | Delta badges | comparación periodo (frontend sobre historial si existe) | omitir badge |
   | Predicción IA del chart | sin endpoint | omitir serie, conservar leyenda solo real |
   | Auto-Piloto banner | `useAgentSettingsQuery` (`mode` si el modelo lo expone) | sugerencia estática + CTA `settings?view=ai` |
3. **Badge de delta** — se renderiza solo si el frontend puede calcular comparación (p. ej. ventas: periodo actual vs anterior sobre el historial de revenue). Si no, sin badge. Nunca valores hardcodeados del mockup.
4. **Rendimiento del equipo** — v1: lista de miembros con barra de "carga de trabajo" derivada de datos disponibles (conversaciones/actividades asociadas) o estado vacío honesto. No inventar %. El diseño muestra % de cumplimiento sin fuente en backend: se sustituye por métricas que el modelo exponga o vacío.
5. **Onboarding helpers (refinado tras revisión)** — `AssistantIntro` + `FirstRunChecklist` se conservan (contratos `ai-onboarding`). Comportamiento verificado: el checklist ya se auto-oculta cuando está completo (`return null`); la novedad es un patrón colapsable que permite **plegar manualmente en estado parcial** (preferencia persistida en localStorage) y, cuando el checklist está completo, conservar un shell mínimo colapsado con la opción de reabrir — sin duplicar ni romper el estado de completitud, que se deriva de las mismas queries existentes en cada render (no hay estado de servidor nuevo). El plegado por defecto NUNCA oculta un checklist incompleto en primer uso.
6. **Acciones rápidas** — se re-mapean a operación: Broadcast → ruta de campañas real si existe (verificar; hoy no hay `/dashboard/campaigns` ni `view=campaigns` en el shell → si no aparece, omitir), Nueva Factura → `settings?view=siigo`, Nuevo Contacto → `/dashboard/crm`, Exportar → `/dashboard/reportes`. Las acciones sin ruta real se omiten (regla D2.2 del shell). **Checkpoint de decisión en tasks 1.6** antes de aplicar.
7. **Copy** — toda la copia nueva (títulos de widgets, estados vacíos, CTA) se agrega a `lib/copy/ui.ts` en español, tipada, sin duplicados con copy existente.
8. **Gates RBAC por widget (herencia de la superficie fuente)** — cada widget SHALL renderizarse solo bajo las mismas condiciones de permiso/entitlement que su superficie fuente; ningún widget amplía acceso al aparecer en home:
   | Widget | Gate heredado |
   |---|---|
   | Conversaciones recientes | misma condición de la bandeja (`useConversationsQuery` accesible al miembro autenticado; el redirect `ORG_MANAGE` de la bandeja queda en la ruta de la bandeja, no se replica en home) |
   | Chart ventas / KPI ventas | `useModule("analytics").enabled` AND `hasPermission(INVOICE_VIEW)` (gate existente en `dashboard-home.tsx`) |
   | Facturas Siigo / KPI facturas | `hasPermission(INVOICE_VIEW)` (+ estado del módulo siigo si existe) |
   | Rendimiento equipo | gate de la superficie de miembros/settings que el modelo exponga (p. ej. members entitlement); si no hay gate explícito, estado vacío honesto, nunca datos de un miembro sin permiso |
   | Banner Auto-Piloto / Copiloto | `useAgentSettingsQuery` disponible al miembro autenticado con acceso a settings de IA (misma condición que `settings?view=ai`) |
   Si una superficie fuente no expone gate, el widget se renderiza con estado vacío honesto, nunca con datos sin permiso.
9. **Minimización de PII en home** — el panel de conversaciones recientes SHALL renderizar únicamente datos a nivel de snippet que la bandeja ya expone (nombre/avatar del contacto, último mensaje truncado, hora relativa, badge de no leídos si el modelo lo expone). NO cuerpos completos de mensajes, NO vista de hilo en home, NO exportación/transferencia nueva de datos personales. La home no crea una superficie de datos mayor que la bandeja; se mantiene la postura de retención/minimización de las specs `data-transfer`/`data-backup-recovery` (Ley 1581/Habeas Data).

## Risks / Trade-offs

- [Widgets sin fuente se ven "vacíos" frente al mockup lleno de cifras] → Mitigación: estados vacíos diseñados con CTA (nunca 0 falsos); documentar en tasks que el llenado real es change futuro de backend. Residual: R3 (perceived-value gap).
- [Rendimiento del equipo sin métrica real] → Mitigación: derivar de datos existentes o vacío honesto; no mostrar % inventados.
- [Reutilizar queries de la home en varios widgets dispara refetch] → Mitigación: TanStack Query cache + `staleTime` existentes; sin fan-out nuevo.
- [Chart con leyenda "predicción IA" sin serie] → Mitigación: solo renderizar leyenda/serie cuando exista la segunda serie; no mostrar línea vacía.
- [Onboarding helpers plegados confunden al primer uso] → Mitigación: plegado por defecto solo cuando el checklist está completo; en estado parcial la preferencia es manual y el estado de completitud se deriva de las queries existentes (nunca se oculta un checklist incompleto).
- [Banner Auto-Piloto podría afirmar estado del agente sin confirmación] → Mitigación: cuando `agent-settings` no expone `mode`, el banner es sugerencia estática (CTA `settings?view=ai`) sin afirmar modo; residual R1 (riesgo de claims de agente autónomo vs términos de Meta).

## Market & Unit Economics

Este change **no altera la economía unitaria del producto**; la declaración es explícita para que el council y el negocio puedan verificarla:

- **Costo de IA por acción: sin delta.** No se añaden acciones de IA, ni llamadas de modelo, ni routing, ni metering nuevo. Los widgets reutilizan queries existentes (`useConversationsQuery`, `useMembersQuery`, `useAgentSettingsQuery`, `analyticsRepository.revenue`) con cache TanStack y sin fan-out nuevo; el banner Auto-Piloto y el panel Copiloto son presentación de estado ya persistido, sin invocación de LLM adicional.
- **Precios / planes / créditos: sin cambio.** No se toca `paywall`, `plan-pricing-ux`, `billing-quota-integrity` ni `ai-usage-metering`. La verificación de pagos Polar/MercadoPago en `app/dashboard/page.tsx` queda byte a byte (verificado). El KPI de ventas usa `analyticsRepository.revenue` con el mismo gate de módulo existente.
- **Margen por plan (Polar + MercadoPago/PSE/Nequi): sin impacto en este change** — no hay flujo de cobro, fee o descuento modificado.
- **Métrica de activación:** el único efecto posible es de presentación (checklist plegado), sin cambio de contrato: la completitud del checklist se sigue derivando de los mismos datos; el plegado manual no borra el progreso. No se introduce ni se mide una métrica nueva de conversión en este change; si el negocio quiere medir impacto de activación de la home recompuesta, se define como follow-up de analytics, no aquí.
- **Regla de honestidad de datos:** el costo de oportunidad de no mostrar las cifras del mockup (1,847 / $47.2M / 284 / 1.2s) es asumido deliberadamente como diferenciador frente a CRMs que inflan KPIs; mitigado por CTAs (R3).

## Market Risk

Riesgos aceptados de este change, con riesgo nombrado, owner y trigger. Ninguno se puede eliminar sin un change futuro de backend o sin decisión de negocio; por eso se registran como residuales:

- **R1 — Deriva de claims de canal/agente (WhatsApp Business Platform).** La home expone datos de conversación (WhatsApp) y un banner Auto-Piloto. El banner NO debe afirmar comportamiento autónomo que la plataforma o `agent-settings` no confirmen (términos de IA de Meta, pricing de conversación, aprobación de templates son superficie de deriva). **Owner:** product/ops. **Trigger:** cambio de términos de Meta sobre agentes de IA o pricing de conversación; o copy del banner que afirme modo autopilot sin confirmación del modelo. **Mitigación en diseño:** banner estático sin afirmar estado cuando no hay `mode` (D2); copy en `lib/copy/ui.ts` sin claims de autonomía.
- **R2 — Regresión de activación por plegado del checklist.** La preferencia de plegado vive en localStorage y puede divergir de la completitud real derivada de datos; un checklist completo auto-ocultado (comportamiento actual verificado) más un shell colapsado podrían reducir visibilidad de onboarding para cuentas que reabren. **Owner:** product. **Trigger:** caída de completitud de pasos de onboarding o de métrica de activación de primer uso (a definir como follow-up de analytics); o reportes de cuentas nuevas que no ven el checklist. **Mitigación:** el plegado por defecto solo aplica cuando el checklist está completo; en primer uso el checklist queda visible (D5).
- **R3 — Gap de valor percibido vs mockup denso (sustitución competitiva: Meta native WhatsApp AI, CRMs locales).** Estados "—"/vacío honestos pueden leerse como producto incompleto en la superficie operativa principal. **Owner:** product/GTM. **Trigger:** feedback cualitativo de clientes o caída de engagement de home; el llenado real (facturas Siigo, tiempo respuesta IA, deltas, predicción IA) se agenda como change futuro de backend. **Mitigación:** CTAs en todos los estados vacíos; la regla de honestidad es diferenciador declarado (Market & Unit Economics).
- **R4 — Expectativa de datos de facturación (ecosistema DIAN/SIIGO).** El widget Facturas Siigo muestra estado vacío con CTA hasta que exista endpoint de lista; los usuarios pueden esperar facturas en home. **Owner:** product/backend. **Trigger:** solicitudes de usuarios de ver facturas desde home, o demanda GTM; el endpoint de lista de facturas se propone en change futuro (no en este). **Mitigación:** estado vacío + CTA `settings?view=siigo`, KPI "—" con hint.

## Migration Plan

1. Implementar widgets y composición en `dashboard-home.tsx` (sustitución del contenido, sin tocar `page.tsx`).
2. Agregar copy a `lib/copy/ui.ts`.
3. Actualizar `dashboard-home.test.tsx` solo si el test depende de estructura/clases (no de lógica de pagos — ese test vive en `app/dashboard/page.test.tsx` y NO cambia).
4. Gates: `pnpm lint`, `pnpm build`, `npx tsc --noEmit`, vitest dashboard-home + page.test.
5. Rollback: git revert; sin estado de DB/Stytch. No hay estado de tenant (Stytch) ni migración que revertir; la única preferencia nueva es localStorage (plegado), que se limpia sola.

## Open Questions

- ¿Existe una ruta real de campañas para "Broadcast"? — **verificado parcialmente**: hoy no hay `/dashboard/campaigns` ni `view=campaigns` en settings; `use-campaign-queries.ts` existe. Resolución en checkpoint de tasks 1.6: si no hay ruta real al implementar, se omite la acción (D2.2).
- ¿El modelo de miembro/conversación permite asociar carga por miembro? (si no, panel vacío honesto).
- ~~¿El checklist de onboarding tiene estado de completitud accesible desde la home?~~ — **RESUELTO (verificado en código)**: sí; el checklist se renderiza en la home y deriva su completitud de queries existentes (`useWhatsAppConfigQuery`, `useSubscriptionQuery`, `useConversationsQuery`, marcadores `ai-onboarding` de localStorage); se auto-oculta al completarse; el colapsable solo añade una preferencia de plegado.
