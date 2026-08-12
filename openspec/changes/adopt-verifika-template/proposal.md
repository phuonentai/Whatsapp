# Proposal: Adopt Verifika/ChatFlow template design site-wide (B-FULL)

## Why

Se aprobó un export de diseño Shuffle (Figma→código) — plantilla **Verifika / ChatFlow CRM** — que ya habla el idioma y el dominio del producto (WhatsApp B2B colombiano: Siigo, DIAN, PSE/Nequi, Ley 1581 Habeas Data, copia en español). Decisión de producto (B-FULL): adoptar la plantilla como **identidad visual y composición** del sitio completo — chrome oscuro `slate-900` + acentos `emerald-500` + superficies de contenido claras — y **extender su lenguaje de diseño a todas las páginas que el zip no cubre** ("missing pages"): las rutas públicas restantes (about, academy, blog, faq, features, plataforma, pricing, legal) y las superficies de producto sin cobertura (CRM, knowledge, procurement, reportes, settings, authenticate). Esto sustituye la dirección "lean soft" de `site-redesign-lean-soft` (23/25 tareas, 2 gates bloqueados por infra), que se archiva con nota de supersedencia. Cero cambios de lógica: auth Stytch B2B, RBAC, billing y datos quedan intactos.

## What Changes

- **Tokens de diseño** (`app/globals.css`, `tailwind.config.ts`): identidad de la plantilla — chrome oscuro `slate-900`/`slate-800` (nav, hero, footer, sidebar, top bar), primario `emerald-500`-family (CTAs, badges, acentos), superficies de contenido claras `slate-50`/`white` con bordes `slate-200`, tipografía display (`font-heading` restaurada para títulos, Inter para texto), idiomas de componente de la plantilla (botones, inputs, tarjetas KPI, badges de estado). Corrección de bug real: `<html lang="en">` → `lang="es"`. Copy español-first se conserva vía `lib/copy/ui.ts`; la copia de la plantilla se fusiona y limpia (tachas tipo "Inciciar session").
- **Landing de marketing** (`app/(marketing)/page.tsx`, `components/marketing/*`): recomposición completa según las 7 secciones de la plantilla — nav → hero con integraciones Meta/Siigo → features ("Integración Oficial Meta & Siigo") → comparación "El Proceso Tradicional vs ChatFlow IA" → stats (conversaciones diarias, horas perdidas, COP en ventas) → pricing → FAQ (Siigo, Ley 1581, costos Meta, número WhatsApp) → footer.
- **Páginas públicas restantes**: `about`, `academy`, `blog`, `faq`, `features`, `plataforma`, `pricing`, `privacy`, `security`, `terms` se re-estilizan al lenguaje de la plantilla (misma chrome + superficies + idioms); rutas, metadata, sitemap y structured data sin cambios.
- **Nueva página `onboarding-info`** (`app/(marketing)/onboarding-info`): "Cómo funciona tu onboarding" — pasos, requisitos, checklist interactivo, cronograma típico y FAQ — contenido traducido/limpio de la plantilla (OnboardingInfoSectionCustomComponents1-7). No existe hoy en el sitio.
- **Auth/onboarding**: `sign-in` usa el layout de la plantilla alrededor de Stytch B2B (magic link, passkeys, MFA TOTP intactos); `signup` adopta la composición del wizard de la plantilla incluyendo pasos fiscales DIAN (NIT, régimen, ciudad) **solo visuales** (sin persistencia nueva); `authenticate` re-estilizado.
- **Superficies de producto**: shell (`components/layout/*`) con chrome oscuro de la plantilla; dashboard por composición de la plantilla (KPI cards + charts) reutilizando `recharts` ya presente y `analytics-repository` existente (RevenueByPeriod, TopCustomersByRevenue, FunnelByStageAggregates — sin endpoints nuevos); inbox adopta la composición de `messages-view` (fila de stats, header con "Nueva campaña" → flujo de campañas existente, lista/thread) con store/filtros/queries intactos; `crm`, `knowledge`, `procurement`, `reportes`, `settings` re-estilizados al lenguaje de la plantilla.
- **Supersedencia**: `site-redesign-lean-soft` se archiva con nota explícita ("dirección revertida por adopt-verifika-template").

## Capabilities

### New Capabilities

- `verifika-visual-identity`: sistema de identidad visual adoptado de la plantilla Verifika/ChatFlow — tokens de color (chrome oscuro slate + acentos emerald + superficies claras), tipografía display + Inter, idioms de componente (botones, inputs, tarjetas KPI, badges, nav) — que gobierna todas las superficies (marketing y producto). Reemplaza la dirección visual de `marketing-site-visual` (delta de `site-redesign-lean-soft`, supersedido).

### Modified Capabilities

- `marketing-website`: la dirección visual cambia de soft-light a la composición de la plantilla (chrome oscuro + emerald + contenido claro); se añade la ruta `onboarding-info`; todas las rutas públicas adoptan el lenguaje de la plantilla manteniendo rutas/metadata/SEO.
- `app-shell`: la requirement de superficie del shell cambia — sidebar/top bar pasan a chrome oscuro de la plantilla (slate-900) en ambos temas (reversión de la dirección soft de lean-soft).
- `dashboard-template-restyle`: la requirement "shell con identidad del template" cambia de identidad — ahora la identidad es la plantilla Verifika (chrome oscuro + emerald), no la paleta empresarial suave.
- `inbox-ui`: la bandeja adopta la composición de `messages-view` (stats, header con "Nueva campaña", layout lista/thread) — comportamiento, store, filtros y queries sin cambios.
- `signup-stytch-compliance`: el wizard de signup adopta la composición de la plantilla con pasos fiscales DIAN visuales (NIT, régimen, ciudad) — sin persistencia nueva, contratos Stytch sin cambios.

## Impact

- **Frontend only**: `next_b2b_starter/` — `app/globals.css` (tokens), `tailwind.config.ts` (fuentes), `components/marketing/*`, `app/(marketing)/*` (incl. nueva ruta `onboarding-info`), `components/layout/*` (shell), `app/dashboard/*`, `app/dashboard/inbox/*`, `app/signup/*`, `app/auth/*`, `app/authenticate/*`, `lib/copy/ui.ts` (fusión de copia de la plantilla). Sin backend, sin migraciones, sin SQLC, sin cambios de API (los charts reutilizan endpoints de analytics existentes).
- **Auth**: cero cambios de contrato Stytch B2B (solo estilos; componentes Stytch intactos); sesiones, RBAC y tenancy sin cambios. Stytch sigue siendo la única autoridad de identidad; PostgreSQL solo almacena `stytch_member_id`/`stytch_organization_id`.
- **Billing**: las ramas de verificación de pago de `app/dashboard/page.tsx` (`checkout_id` Polar, `payment_id`/`preapproval_id` MercadoPago) se conservan byte a byte en comportamiento.
- **Dependencias**: ninguna nueva. `react-apexcharts` de la plantilla NO se adopta (se usa `recharts` ya presente); assets stock de la plantilla (logos `plain-*`, avatares) solo como placeholders cuando aplique.
- **SEO/perf**: rutas y metadata preservadas; `lang="es"` es una mejora real (a11y/SEO); `onboarding-info` añade metadata propia; el chrome oscuro no añade coste de pintura significativo.
- **Ops**: `pnpm build`, `pnpm lint`, `npx tsc --noEmit`, gates de vitest existentes (dashboard, signup, inbox, dashboard-home, plans-modal); visual/a11y Playwright en 390x844/768x1024/1440x900 → `qa/`; e2e de retorno de pago sigue bloqueado por infraestructura (misma pared que lean-soft; cubierto por unit tests de routing 10/10).
- **Rollback**: git revert del change; no hay estado de DB ni de Stytch que revertir; los tokens anteriores son recuperables del historial.
- **Non-Goals**: sin cambios de rutas/SEO-estructura (excepto añadir `onboarding-info`); sin reescritura de lógica de negocio; sin persistir datos fiscales DIAN (solo visual; change futuro ligado a siigo-invoicing); sin endpoints nuevos de backend; sin eliminar el soporte de tema oscuro (la identidad ES oscura); sin migrar a CMS; sin almacenar credenciales localmente (todo auth sigue en Stytch B2B); sin rediseñar contenido interno de settings/CRM/reportes/knowledge más allá de los tokens y el lenguaje de la plantilla.

## Assumptions

- "Missing pages" = todas las superficies sin cobertura de la plantilla (rutas públicas restantes + superficies de producto restantes + `authenticate`); se crean/re-estilizan con el lenguaje de diseño de la plantilla, no se eliminan.
- `messages-view` de la plantilla se pliega en la ruta existente `app/dashboard/inbox` (no se crea ruta nueva); `onboarding-info` sí es una ruta nueva porque no tiene equivalente.
- La paleta de la plantilla (slate-900 + emerald-500 + slate-50/white) se adopta tal cual como identidad; los valores HSL exactos se fijan en design.md y pueden ajustarse en implementación sin cambiar el contrato de spec.
- La adopción es B-FULL: composición y lenguaje de la plantilla en todas las superficies, re-construidos sobre la app real (datos, RBAC, billing), no copia literal de mockups estáticos.
- `site-redesign-lean-soft` queda supersedido y se archiva con nota (tarea OPS-GOV); sus gates 4.3/4.4 bloqueados por infra se heredan como riesgo conocido.
