# Design: Adopt Verifika/ChatFlow template (B-FULL)

## Context

- **Template**: export Shuffle (Figma→código) `next_b2b_starter/website/shuffle-20260812-0417-14431.zip` — plantilla "Verifika / ChatFlow CRM": Next 15 + Tailwind 4 + react-apexcharts, pages router, copia en español (con typos), dominio WhatsApp B2B colombiano (Siigo, DIAN, PSE/Nequi, Ley 1581). Páginas: index (7 secciones), sign-in, onboarding, onboarding-info, messages-view, dashboard.
- **App real**: `next_b2b_starter/` — Next 16, app router, shadcn/ui, TanStack Query, server actions, copy tipada español-first (`lib/copy/ui.ts`), auth Stytch B2B, billing Polar/MercadoPago, RBAC.
- **Estado de specs**: `app-shell` y `dashboard-template-restyle` ya mandan chrome oscuro `slate-900` (identidad del template original). `site-redesign-lean-soft` (23/25, in-progress) revirtió a soft-light — se supersede y archiva con nota.
- **Hallazgo clave**: la plantilla Verifika y la app real ya comparten la misma anatomía — sidebar con grupos (Dashboard, Conversaciones, Contactos, Facturas, Pagos, Analíticas; grupo IA con Copiloto IA/Entrenamiento/Automatizaciones), tarjeta "IA Insights", top bar con búsqueda ⌘K/notificaciones/usuario, KPIs (conversaciones activas, ventas semana, facturas emitidas, tiempo respuesta IA), chart de ventas con selector de periodo, panel "Copiloto IA", conversaciones recientes. El port es **restyle + rewire**, no invención.

## Goals / Non-Goals

**Goals:**
- Adoptar la identidad Verifika (chrome oscuro slate-900 + emerald + contenido claro) en todas las superficies (B-FULL).
- Re-componer la landing según las 7 secciones de la plantilla; re-estilizar todas las rutas públicas restantes; añadir `/onboarding-info`.
- Portar las composiciones de producto (dashboard con charts, bandeja messages-view, wizard de signup con pasos DIAN visuales) sobre la app real.
- Cero cambios de lógica: Stytch, RBAC, billing, datos, rutas.

**Non-Goals:**
- Sin endpoints nuevos de backend (los charts reutilizan analytics existentes + recharts).
- Sin persistencia de datos fiscales DIAN (solo visual; follow-up ligado a siigo-invoicing).
- Sin adoptar react-apexcharts ni assets stock de marca de la plantilla.
- Sin cambios de rutas existentes ni de SEO-estructura (excepto añadir `/onboarding-info`).

## Decisions

### D1. Tokens: identidad Verifika sobre el sistema de temas existente

La paleta se adopta como: chrome oscuro fijo `slate-900`/`slate-800`/`slate-700` (nav, hero, sidebar, top bar, footer) con utilidades explícitas; superficies de contenido con tokens shadcn del tema (claro: `slate-50`/`white`, bordes `slate-200`; oscuro suave: tokens `.dark` existentes) + acentos `emerald-500`-family en utilidades. **Por qué**: el chrome fijo ya está mandado en `app-shell`/`dashboard-template-restyle`; los tokens de contenido preservan el toggle de tema y el trabajo previo. **Alternativas**: tokenizar el emerald como `--primary` (rechazado: el primary en tokens afecta contratos de contraste/estados de otros componentes; la plantilla usa utilidades explícitas — mismo approach que el restyle anterior).

### D2. Tipografía: restaurar Sora para `font-heading`

Se restaura la fuente display vía `next/font/google` (Sora) bajo alias `font-heading` en `tailwind.config.ts`; Inter sigue para body. La plantilla usa `font-heading` 64+ veces sin definición propia — el alias resuelve en la app real. **Por qué**: la identidad display es parte del ADN de la plantilla (lean-soft la retiró; B-FULL la restaura). **Alternativa**: Inter-only (rechazado — pierde la identidad display del hero/títulos).

### D3. Charts: recharts + analytics-repository existentes

El chart "Rendimiento de Ventas" (selector 7d/30d/mes/trimestre, ventas reales vs predicción IA) se implementa con `recharts` (ya en `package.json`, ya usado en `dashboard-home.tsx` y `reportes-page.tsx`) sobre `analytics-repository` (RevenueByPeriod, TopCustomersByRevenue, FunnelByStageAggregates — backend Go ya expone estos queries org-scoped). La "predicción IA" sin fuente de datos real SHALL renderizarse como serie vacía/placeholder ("—"), nunca cifras inventadas. **Por qué**: cero dependencias nuevas, cero endpoints nuevos. **Alternativa**: adoptar react-apexcharts de la plantilla (rechazado — dependencia nueva, stack duplicado, sin ventaja).

### D4. Composición de marketing: secciones de plantilla como componentes

Las 7 secciones de la plantilla (nav, hero+integraciones, features "Integración Oficial Meta & Siigo", comparación "Tradicional vs ChatFlow IA", stats, pricing, FAQ, footer) se portan como componentes React en `components/marketing/*` (app router, server components donde no haya interactividad; client solo donde haga falta: FAQ accordion, mobile menu), con copy desde `lib/copy/ui.ts` (merge de la copia de la plantilla, typos corregidos). Las rutas restantes (`about`, `academy`, `blog`, `faq`, `features`, `plataforma`, `pricing`, `privacy`, `security`, `terms`) reutilizan primitivas compartidas (`page-hero`, `section-heading`) re-estilizadas a la identidad. `/onboarding-info` es una ruta nueva con las 7 sub-secciones de `OnboardingInfoSectionCustomComponents1-7` (pasos, requisitos, checklist, cronograma, FAQ). **Por qué**: composición fiel sin reescribir el sistema de rutas/SEO existente.

### D5. messages-view → se pliega en `app/dashboard/inbox`

No se crea ruta nueva: la composición de messages-view (header "Mensajes" + Conectado + "Nueva campaña", stats cards, lista) se implementa dentro de la bandeja existente, que ya tiene store/queries/filtros/tests. "Nueva campaña" enlaza al flujo de campañas existente. **Por qué**: conserva URL contracts, RBAC y tests; la plantilla es un mockup estático sin lógica propia. **Alternativa**: ruta `/dashboard/messages` (rechazado — duplica la bandeja y rompe la navegación actual).

### D6. Wizard de signup: composición Verifika + pasos DIAN visuales

El wizard de la plantilla (bienvenida "Configura tu empresa en 3 minutos", empresa: nombre/NIT/régimen/ciudad, Conecta WhatsApp: migrar/nuevo número +57) se implementa sobre `use-signup-flow.ts` y los componentes Stytch existentes. Los campos fiscales DIAN son **solo visuales**: no se añaden campos al payload, no hay persistencia local. **Por qué**: el contrato Stytch (`SendInvite: true`, sin `owner_password`, códigos de error estructurados) y el signup real no deben cambiar; la recolección de datos fiscales es un cambio de producto futuro ligado a siigo-invoicing (Open Question). **Alternativa**: persistir NIT/régimen/ciudad (rechazado en este change — requiere migración SQL + DTOs + validación; se separa como change futuro).

### D7. Shell: restyle al chrome Verifika, navegación y RBAC intactos

Sidebar/top bar: `slate-900` + bordes `slate-800`, ítem activo con acento emerald, textos `slate-300/400` — exactamente la anatomía que la plantilla y la app ya comparten; se conservan rutas, grupos RBAC-filtered, tarjeta "IA Insights" con render condicional, búsqueda ⌘K, notificaciones y menú usuario. **Por qué**: la navegación filtrada por permisos ya existe y es un contrato; el cambio es puramente de utilidades de clase.

### D8. lang="es" + copy español-first

`app/layout.tsx` cambia `lang="en"` → `lang="es"` (bug real de a11y/SEO). La copia de la plantilla se fusiona en `lib/copy/ui.ts` (namespace `marketing` + nuevos keys de onboarding-info), corrigiendo typos ("Inciciar session" → "Iniciar sesión").

### D9. Supersedencia de site-redesign-lean-soft

Se archiva `site-redesign-lean-soft` con nota explícita de supersedencia (dirección revertida). Sus gates 4.3/4.4 (e2e retorno de pago, Playwright) bloqueados por infra se heredan como riesgo conocido de este change.

## Risks / Trade-offs

- **[Conflictos con 6 changes in-flight]** passkeys, MFA, MercadoPago, client-payments, billing-lifecycle, scheduled-inquiry-runs tocan los mismos shells (auth/signup/dashboard/inbox) → Mitigación: este change es estilos sobre lógica; se ejecuta después o se resuelven conflictos por diffs de clases, nunca de lógica. Los gates de verificación del change cubren las superficies tocadas.
- **[e2e retorno de pago bloqueado por infra]** backend Go sin secretos en este entorno → Mitigación: unit tests de routing 10/10 ya cubren las ramas; el restyle conserva las ramas byte a byte en comportamiento; se registra el gate manual como pendiente (misma pared que lean-soft).
- **[Charts sin datos → placeholders]** la "predicción IA" no tiene fuente → Mitigación: serie vacía + "—"; el chart real (RevenueByPeriod) usa el endpoint existente.
- **[Copia de plantilla con typos/marca ajena]** "Verifika"/"ChatFlow" y typos → Mitigación: merge a copy layer con marca NexoChat y corrección ortográfica (requisito de spec).
- **[Assets stock]** logos plain-*/avatares genéricos → Mitigación: solo placeholders decorativos; la marca NexoChat prevalece.

## Migration Plan

1. **Fases de implementación** (tasks.md): A) tokens/tipografía/lang → B) landing + rutas públicas + `/onboarding-info` → C) auth/signup → D) shell/producto (dashboard, inbox, superficies restantes) → E) verificación (unit, visual/a11y, SEO) + archivado de lean-soft.
2. **Deploy**: `pnpm build`/`lint`/`tsc` gates en cada fase; e2e Playwright visual en 390/768/1440 → `openspec/changes/adopt-verifika-template/qa/`.
3. **Rollback**: git revert del change. Sin estado de DB ni de Stytch que revertir (visual-only); los tokens anteriores son recuperables del historial.

## Open Questions

- **Datos fiscales DIAN**: ¿se recolectan NIT/régimen/ciudad en un change futuro ligado a `add-siigo-invoicing`? (Este change los mantiene visuales; se propone follow-up.)
- **Predicción IA del chart**: ¿hay fuente de datos planeada (módulo de predicción) o queda como placeholder "—" hasta que exista?
- **Assets de la plantilla**: ¿se desean conservar algunos (avatares, map.png) como decorativos o se descartan todos en favor de los existentes de la app?
