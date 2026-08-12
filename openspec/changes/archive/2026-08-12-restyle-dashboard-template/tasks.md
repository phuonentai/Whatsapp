## 1. Dashboard shell [FE-NEXT]

- [x] 1.0 Traducir etiquetas del shell al español vía copy layer: "Dashboard"→"Panel", "Inbox"→"Bandeja", "Knowledge Base"→"Base de conocimiento", "Settings"→"Configuración" (actualizar `components/layout/sidebar.tsx` y `header.tsx`, incluyendo aria-labels "Open sidebar"/"Close sidebar"→español). Verify: `pnpm build`; sidebar 100% español
  - Hecho: nuevo namespace `layout` en `lib/copy/ui.ts` (es/en); sidebar/header usan `ui.layout.*` (navDashboard→Panel, navConversations→Conversaciones, navSettings→Configuración, aria-labels openSidebar/closeSidebar/expandSidebar/collapseSidebar). La etiqueta del item de bandeja usa la composición del template ("Conversaciones"); "Bandeja" se mantiene como título de la página (`ui.inbox.title`).
  - Verify: `pnpm build` PASS · `npx tsc --noEmit` PASS

- [x] 1.1 Restyle `components/layout/dashboard-layout.tsx` + sidebar: fondo `bg-slate-900` con grupos de navegación (Dashboard, Conversaciones, Contactos, Facturas, Pagos, Analíticas; grupo "Inteligencia Artificial": Copiloto IA, Entrenamiento, Automatizaciones) y tarjeta "IA Insights" al pie; logo NexoChat (reusar `LogoMark` de `components/marketing/site-header.tsx`). **Los grupos nuevos y la tarjeta "IA Insights" SHALL integrarse al modelo existente de navegación filtrada por permisos/entitlement del sidebar** (`permissions.permissions.includes(item.permission)`): ítems sin permiso o sin ruta real NO se enlazan incondicionalmente. Mantener rutas/perfiles reales del sidebar actual. Verify: `pnpm dev` muestra el shell oscuro; navegación existente intacta; grupos del template solo visibles según permisos
  - Hecho: `sidebar.tsx` restilizada (bg-slate-900, LogoMark, grupos del template, tarjeta IA Insights). Mapeo a rutas reales con gating existente: Panel(/dashboard, sin gate) · Conversaciones(/dashboard/inbox, org:manage) · Contactos(/dashboard/crm, entitlement crm_*) · Facturas(/dashboard/settings?view=siigo, org:manage) · Pagos(/dashboard/settings?view=subscription, org:manage) · Analíticas(/dashboard/reportes, entitlement analytics_module) · Base de conocimiento(/dashboard/knowledge) · Proveedores(/dashboard/procurement, org:manage) · Configuración(/dashboard/settings). Grupo IA: Copiloto IA(/dashboard/settings?view=ai, org:manage) + tarjeta IA Insights (CTA→view=ai, gated org:manage). **Decisión documentada (D2.2): Entrenamiento y Automatizaciones NO se renderizan — no existe ruta real en el repo; no se enlazan incondicionalmente.** Activo de items con query (settings views) vía `useSearchParams` (layout dinámico → seguro).
  - Verify: `pnpm build` PASS (shell compila) · smoke dev pendiente de infraestructura (dev server compartido con error Turbopack worker-spawn, ver 5.1)

- [x] 1.2 Top bar: búsqueda (⌘K → command palette existente), notificaciones, avatar de usuario con plan (usar datos de AuthProvider/entitlement reales; sin fabricar nombres — mostrar iniciales del perfil). Verify: `pnpm lint`, `pnpm build`
  - Hecho: `header.tsx` oscuro (slate-900, borde slate-800); búsqueda ⌘K (palette existente, `ui.layout.searchPlaceholder`); campana de notificaciones → `/dashboard/settings?view=audit` (superficie real de actividad/auditoría; sin UI de notificaciones inventada); soporte/preferencias/avatar con iniciales reales del perfil (`user-menu.tsx` oscuro, círculo emerald, nombre/plan reales de AuthProvider).
  - Verify: `pnpm lint` PASS (0 errores / 4 warnings pre-existentes) · `pnpm build` PASS

- [x] 1.3 Mantener next-themes funcional (contenido respeta el tema; shell fijo oscuro `slate-900` según delta app-shell). Verify: toggle de tema sigue operativo en páginas de contenido
  - Hecho: shell/sidebar/top bar usan utilidades explícitas slate/emerald (fijo oscuro); contenido conserva tokens de tema (bg-card, border-border, texto foreground); el toggle de tema sigue en el UserMenu (`useTheme`/next-themes intacto).
  - Verify: `pnpm build` PASS · smoke dev pendiente de infraestructura

## 2. Dashboard overview real [FE-NEXT]

- [x] 2.1 `app/dashboard/page.tsx`: **conservar sin cambios las ramas server-side de verificación de parámetros de pago** (Polar `checkout_id` → `verifyPayment`; MercadoPago `payment_id`/`preapproval_id` → `verifyMercadoPagoPayment`) y sus redirects a `/dashboard/settings?view=subscription&payment_{verified,error}=true`; sobre `DashboardHome` (NO es un redirect: ya renderiza KPIs, quick actions, `FirstRunChecklist`, `AssistantIntro`) componer el overview del template: saludo + selector de rango (si existe analytics) + botón "Nueva Conversación" (→ /dashboard/inbox). Reutilizar/restilizar el contenido existente de `DashboardHome`, no reemplazarlo. Verify: `/dashboard` renderiza overview; `pnpm exec playwright test` del spec de retorno de checkout (o manual) pasa
  - Hecho: `app/dashboard/page.tsx` NO se tocó — ramas de verificación intactas (verificado: `verifyPayment`/`verifyMercadoPagoPayment` con redirects; `app/dashboard/page.test.tsx` pasa 4/4, cubre preapproval-only, payment_id, checkout Polar y error). Overview compuesto en `DashboardHome`: saludo con nombre real del perfil, selector Semana/Mes (solo si analytics disponible), botón "Nueva Conversación"→/dashboard/inbox; contenido existente (checklist, intro asistente, deals por etapa, actividad reciente, quick actions) conservado debajo.
  - Verify: `npx vitest run app/dashboard/page.test.tsx` PASS (4/4) · `pnpm build` PASS · retorno de checkout e2e pendiente de infraestructura (ver 5.1)

- [x] 2.2 4 tarjetas KPI alimentadas por las queries que YA consume `DashboardHome` (`useConversationsQuery` + queries CRM/analytics según existan; sin fan-out nuevo); valor "—" cuando no haya fuente. Verify: KPIs muestran datos reales o "—", nunca cifras inventadas
  - Hecho: Conversaciones activas (real, `useConversationsQuery`); Ventas semana (real vía `analyticsRepository.revenue({period})` solo si módulo analytics + permiso invoice:view; "—" si no); Facturas emitidas → "—" (sin fuente: siigo expone status/numeration, no conteo de facturas); Tiempo respuesta IA → "—" (sin query de métricas de agente). Sin queries nuevas fuera de las existentes (revenue es la misma del módulo reportes).
  - Verify: `pnpm build` PASS

- [x] 2.3 Gráfico de rendimiento de ventas (datos analytics si existen; estado vacío con CTA a reportes) + panel "Copiloto IA" (insights del módulo agent si existen; si no, CTA a configurar). Verify: `pnpm build`, e2e dashboard (si existe) pasa
  - Hecho: gráfico recharts con revenue analytics (barras emerald) cuando módulo+permiso; estado vacío con CTA a `/dashboard/reportes` si no; panel "Copiloto IA" oscuro (gradiente slate-900→800) con 3 insights orientativos + CTA a `/dashboard/settings?view=ai` (config real del asistente).
  - Verify: `pnpm build` PASS

## 3. Inbox restyle (messages-view) [FE-NEXT]

- [x] 3.1 `app/dashboard/inbox/*`: aplicar composición messages-view — 4 tarjetas de métricas (conversaciones hoy, por responder, tasa de respuesta, tiempo promedio) con datos del inbox store existente; toolbar con búsqueda + filtros estado/agente (usar estados/agentes reales de la lógica actual); lista de conversaciones con avatar (iniciales), etiqueta (Cliente/Prospecto según CRM tags existentes), snippet, hora, badge no leídos. Verify: `pnpm dev` bandeja operativa con datos reales; `pnpm lint`
  - Hecho: nuevo `components/inbox-metrics.tsx` — Conversaciones hoy (lastMessageAt hoy, real) y Por responder (activas con no-leídos o sugerencia pendiente, real); Tasa de respuesta y Tiempo promedio → "—" (sin fuente de datos; regla del spec). `conversation-list.tsx` restilizada a messages-view: avatar iniciales, badge canal, nombre+hora, chip de estado (Activa/Cerrada/Archivada), línea de subtítulo (teléfono/@usuario), badge violeta de sugerencias pendientes y punto emerald de no leídos; tabs de canal/estado con lógica existente. Toolbar con filtros reales (canal + estado; sin agente porque el modelo de conversación no expone agente).
  - Verify: `pnpm lint` PASS · `npx vitest run app/dashboard/inbox/components/conversation-list.test.tsx` PASS (3/3) · `pnpm build` PASS · smoke dev pendiente de infraestructura

- [x] 3.2 Actualizar page-objects de e2e si dependen de clases/textos del inbox (sin cambiar lógica). Verify: `pnpm exec playwright test inbox-ui` (o el spec equivalente) pasa
  - Hecho: `e2e/page-objects/admin-panel.page.ts` — mapeo sidebar español ("Inbox"→"Conversaciones", "CRM"→"Contactos"); `e2e/page-objects/inbox.page.ts` — `assertConversationStatus` traduce status → etiqueta (closed→"Cerrada", archived→"Archivada"). Lógica no tocada; specs sin cambios.
  - Verify: e2e completo pendiente de infraestructura (backend Go + dev server funcional, ver 5.1)

## 4. Onboarding/signup restyle [FE-NEXT]

- [x] 4.1 `app/signup/*`: wizard oscuro según onboarding.html (progreso, prev/next, tarjetas de tipo de negocio con los valores actuales del flujo) sobre `use-signup-flow.ts` sin cambios de lógica. Verify: flujo account→organization→business completo en `pnpm dev`; magic link Stytch intacto
  - Hecho: `app/signup/page.tsx` — wizard oscuro (bg gradient slate-900→950, card slate-900, barra de progreso "Paso X de 3" + barra emerald, botones prev outline / next emerald, inputs oscuros); vista email-sent también oscura. `business-context-step.tsx` — tarjetas de selección oscuras (border emerald cuando seleccionada). Placeholders/textos de e2e conservados (Juan Pérez, tu@empresa.com, Acme S.A.S., select industria, "Continuar", "Crear cuenta", objetivo de negocio). `use-signup-flow.ts` sin cambios.
  - Verify: `npx vitest run app/signup/page.test.tsx` PASS (3/3) · `pnpm build` PASS · smoke dev pendiente de infraestructura

- [x] 4.2 Verificar contrato Stytch sin cambios: mismas llamadas (`sendMagicLink`), mismos redirects. Verify: `pnpm exec playwright test auth-passwordless` (o spec equivalente) pasa
  - Hecho: `use-signup-flow.ts` intacto (llamadas y redirects idénticos); `BusinessContextStep` conserva `handleContinue → saveBusinessContext + onContinue(sendMagicLink)`; sin cambios en componentes Stytch.
  - Verify: `pnpm build` PASS · e2e auth-passwordless pendiente de infraestructura (ver 5.1)

## 5. Verificación y cierre [OPS-GOV]

- [x] 5.1 Gate de verificación registrado:
  - `pnpm lint` — **PASS** (0 errores / 4 warnings pre-existentes según baseline)
  - `pnpm build` — **PASS** (Next 16.0.10, producción; todas las rutas compilan)
  - `npx tsc --noEmit` — **PASS**
  - Unit tests — **PASS**: `npx vitest run app/dashboard/page.test.tsx app/signup/page.test.tsx app/dashboard/inbox/components/conversation-list.test.tsx` → 10/10 (incluye routing de checkout Polar/MercadoPago y página de signup)
  - `openspec validate restyle-dashboard-template` — **PASS**
  - Smoke `pnpm dev` (/dashboard, /dashboard/inbox, /signup) — **PASS (2026-08-12, servidor sano puerto 3100)**: `/signup` → HTTP 200 con wizard oscuro SSR (markup slate-900 presente); `/dashboard` y `/dashboard/inbox` → 307 al middleware de auth (`/auth?returnTo=...`), comportamiento esperado sin sesión Stytch. El servidor compartido 3001 sigue fallando con `TurbopackInternalError: spawning node pooled process - No such file or directory` en TODAS las rutas (incluidas las no tocadas) — limitación del entorno documentada; `pnpm build` PASS valida compilación y render server-side. (Nota: `/auth` devuelve 500 en el dev server por la misma inestabilidad Turbopack; ruta no tocada por este change.)
  - Retornos de checkout (Polar `checkout_id` / MercadoPago `payment_id`/`preapproval_id`) — **PENDIENTE de infraestructura**: requiere stack completo (backend Go + webhooks). Cubiertos por `app/dashboard/page.test.tsx` (unit, 4/4) que valida las ramas intactas.
  - e2e Playwright (inbox-ui, auth-passwordless, admin-panel) — **PENDIENTE de infraestructura**: requiere backend Go + dev server funcional; page-objects actualizados en 3.2.

### Re-verificación gate (2026-08-12, segunda corrida — estado actual del entorno)

  - `pnpm lint` — **PASS** (0 errores / 4 warnings pre-existentes, mismas 4)
  - `pnpm build` — **PASS** (producción; todas las rutas compilan, incl. `/dashboard`, `/dashboard/inbox`, `/signup`)
  - `npx tsc --noEmit` — **PASS**
  - Unit tests — **PASS 10/10** (dashboard 4/4, signup 3/3, conversation-list 3/3)
  - `openspec validate restyle-dashboard-template` — **PASS**
  - Smoke dev — **PASS (2026-08-12, servidor sano puerto 3000)**: `/signup` → HTTP 200 con wizard oscuro SSR (markup `bg-gradient-to-b from-slate-900 to-slate-950`, `bg-slate-900`, "Paso", "Continuar" presentes); `/dashboard` y `/dashboard/inbox` → 302 a `/auth?returnTo=...` (middleware de auth, esperado sin sesión Stytch); `/auth` → 200. Puerto 3100: `/signup` 200 pero `/auth` 500 (inestabilidad Turbopack documentada en rutas no tocadas; el puerto 3000 sano cubre el smoke).
  - **Nuevo hallazgo — blocker de e2e identificado (2026-08-12):** el backend Go NO arranca: panic en bootstrap de DI — `internal/modules/procurement/app/services/subscriber.go:28: missing type: services.MetricsSink` (init `internal/modules/procurement/cmd/init.go:47`). Es un fallo **pre-existente del árbol** (módulo procurement de un change hermano en vuelo; `go build ./...` del Phase 0 baseline pasa, pero el wiring runtime falla). Fuera del scope de este change (FE-only, Non-Goals: "no tocar el backend"): NO se corrige aquí; se registra como bloqueador concreto del gate. Con el backend caído no es posible correr e2e (X-Test-Org-ID + `/api`) ni retornos de checkout en stack completo.
- [x] 5.2 Decisión de archive: `/opsx-archive` o `**Archive deferred:** <razón>` en tasks.md
  - **Council re-aprobado (2026-08-12):** re-revisión tras el rework del diseño — `VERDICT.md` ahora es `STATUS: APPROVED` (los 4 cambios requeridos del veredicto previo están resueltos: ramas de pago preservadas + unit 4/4, checks de retorno de checkout en el gate, navegación RBAC, sin fan-out con regla "—").
  - **Finding A resuelto (2026-08-12):** delta spec `specs/dashboard-template-restyle/spec.md` (requirement "Bandeja con composición messages-view" + scenario) enmendado — filtros por **canal y estado** (agente solo si el modelo de datos lo expone; `Conversation` no lo expone hoy); `proposal.md` alineado; living spec re-sincronizado (`openspec/specs/dashboard-template-restyle/spec.md`, requirements idénticos verificados por diff). `openspec validate restyle-dashboard-template` PASS. Condición #1 del veredicto cumplida.
  - **Archive deferred (re-confirmado 2026-08-12, segunda corrida):** el gate de verificación (AGENTS.md) sigue con pasos PENDIENTES de infraestructura y el bloqueador concreto ahora está identificado: el backend Go no arranca por un panic DI pre-existente en el módulo `procurement` (sibling change; fuera de scope FE-only — ver hallazgo en 5.1). Sin backend no corren e2e Playwright (inbox-ui, auth-passwordless, admin-panel) ni los retornos de checkout Polar/MercadoPago en stack completo. Los checks ejecutables (lint/build/tsc/unit 10/10/openspec validate/smoke puerto 3000) se re-confirmaron PASS en esta corrida. Re-archivar cuando el backend arranque y el gate esté completo.

## Phase 0 baseline checkpoint (2026-08-11, repo-wide active-changes run)

- [x] Repo-wide baseline recorded BEFORE further implementation work on this change (working tree: ~330 modified files across both apps from sibling in-flight changes):
  - `go build ./...` PASS (exit 0) · `go vet ./...` PASS · `go test ./...` PASS (all packages, exit 0) — go-b2b-starter
  - `npx tsc --noEmit` PASS · `pnpm lint` PASS (0 errors / 4 pre-existing warnings) · `pnpm build` PASS — next_b2b_starter
  - Context: this baseline anchors later verification gates — failures introduced by this change are distinguishable from pre-existing tree state.
