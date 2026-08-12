# Proposal: Rediseño del sitio — más ligero, colores empresariales suaves

## Why

Un equipo de UI/UX, SEO y diseño revisó la página actual y la considera pesada: hero y secciones oscuras `slate-900`, acento `emerald-500` saturado, tipografía display (Sora), motion decorativo y muchas secciones densas en la landing. La directiva es **hacer el sitio más ligero (leaner)** y usar **colores empresariales suaves** (neutros cálidos, azul corporativo apagado, verdes salvia) para transmitir confianza B2B, mejorar legibilidad y reducir la fatiga visual — sin perder la identidad NexoChat (WhatsApp, Siigo, MercadoPago) ni los contratos de negocio.

El problema es de presentación, no de contenido ni de arquitectura: rutas, copy, datos, auth Stytch, RBAC y billing quedan intactos. Esto alinea también la coherencia marca↔producto: el shell oscuro fijo del dashboard (heredado del restyle al template) se sustituye por superficies suaves con los nuevos tokens, manteniendo la verificación de pagos y toda la lógica existente.

## What Changes

- **Tokens de diseño** (`app/globals.css`, `tailwind.config.ts`): paleta shadcn suave — neutros cálidos (fondo `hsl` cercano a `#FAFAF8`, texto gris-pizarra suave `#1F2937`, bordes `#E5E7EB`), primario azul corporativo apagado (desaturado, e.g. `#3B6CB8`-family con contraste AA), secundario verde salvia suave; se conserva el sistema de temas claro/oscuro (variante oscura suave, no negro puro). La tipografía pasa a **solo Inter** (se retira Sora/font-heading): look empresarial limpio, más ligero.
- **Sitio de marketing** (`app/(marketing)/*`, `components/marketing/*`): recomposición ligera sobre las mismas rutas y copy —
  - Secciones oscuras `slate-900` → superficies claras suaves (`bg-white`/`slate-50`, secciones alternadas con bordes sutiles); gradientes emerald pesados eliminados.
  - Hero simplificado: menos decoración, badge + título + lead + 2 CTAs + proof points discretos; sin gradientes de fondo.
  - Composición reducida: logo-strip, feature grid 4-col, comparison, pricing 4-plan, FAQ, CTA y footer se mantienen pero con menor peso visual (menos sombras, menos padding heroico, tipografía más pequeña y calma, `rounded` suaves).
  - Motion: `Reveal`/framer-motion se reduce a un fade-up sutil o se desactiva en rutas críticas (SEO/perf: sin scroll-jacking, sin animaciones que bloqueen LCP).
  - Accentos emerald → azul corporativo apagado con verde salvia como acento secundario (identidad WhatsApp/Siigo preservada con croma bajo).
  - Botones: emerald-500 saturado → primario azul suave; sombras grandes (`shadow-emerald-500/25`) eliminadas.
- **Shell del dashboard** (`components/layout/*`): sidebar/top bar oscuro fijo `slate-900` → superficies claras suaves con tokens de tema (ambos temas); navegación, permisos RBAC y rutas sin cambios. La tarjeta "IA Insights" y grupos "Inteligencia Artificial" se re-estilizan sin cambiar su lógica de render condicional.
- **Superficies del producto** (overview `app/dashboard/page.tsx`, bandeja `app/dashboard/inbox/*`, signup `app/signup/*`): se actualizan las utilidades explícitas `slate-*`/`emerald-*` a los nuevos tokens suaves. **Sin cambios de comportamiento**: verificación de parámetros de pago (Polar `checkout_id`, MercadoPago `payment_id`/`preapproval_id`) intacta; flujo Stytch y `use-signup-flow.ts` intactos; KPIs con "—" cuando no hay fuente.

## Capabilities

### New Capabilities

- `marketing-site-visual`: presentación visual del sitio público — paleta empresarial suave, composición ligera (hero, feature grid, pricing, FAQ, footer), tipografía Inter-only, motion restringido; cubre las rutas `app/(marketing)/*` y `components/marketing/*`. (El sitio de marketing hoy no tiene spec que lo gobierne; este change lo introduce.)

### Modified Capabilities

- `dashboard-template-restyle`: la requirement "Dashboard shell con identidad del template" cambia — el shell pasa de sidebar/top bar `slate-900` fijo a superficies suaves con tokens de tema; overview/bandeja/onboarding adoptan la paleta suave manteniendo todos los contratos de comportamiento (verificación de pagos, filtros, wizard Stytch).
- `app-shell`: la requirement de dark mode cambia — el shell ya NO renderiza superficie fija `slate-900`; sidebar y top bar siguen los tokens del tema (superficies suaves en claro y oscuro), manteniendo persistencia y toggle.

## Impact

- **Frontend only**: `next_b2b_starter/` — `app/globals.css` (tokens), `tailwind.config.ts` (fuentes/tokens), `components/marketing/*` (recomposición), `app/(marketing)/*` (layout/páginas, sin cambios de ruta), `components/layout/*` (shell), y actualizaciones puntuales de clase en `app/dashboard/page.tsx`, `app/dashboard/inbox/*`, `app/signup/*`. Sin backend, sin migraciones, sin SQLC, sin cambios de API.
- **Auth**: cero cambios de contrato Stytch (wizard y componentes intactos; solo estilos).
- **Billing**: las ramas de verificación de pago de `app/dashboard/page.tsx` se conservan byte a byte en comportamiento; se añade verificación explícita del flujo de retorno de checkout al gate.
- **SEO**: rutas, metadata y structured data no cambian (el rediseño es visual); se elimina cualquier animación que pudiera impactar LCP. **Perf**: menos gradientes/sombras → menor coste de pintura.
- **Dependencias**: ninguna nueva. Sora se retira (se elimina `next/font/google` de Sora; Inter ya está); framer-motion se conserva como dependencia pero con uso mínimo.
- **Ops**: `pnpm build`, `pnpm lint`, `pnpm dev` en `next_b2b_starter/`; e2e Playwright existentes deben seguir pasando (si un page-object depende de clases de estilo, se actualiza; los selectores de lógica no cambian).
- **Rollback**: git revert del change. No hay estado de DB ni de Stytch que revertir; los tokens anteriores son recuperables del historial.
- **Non-Goals**: sin cambios de rutas/SEO-estructura; sin reescritura de copy (solo se recorta presentación); sin nuevos features; sin tocar backend, auth, RBAC, billing ni datos; sin eliminar el soporte de tema oscuro (variante oscura suave); sin migrar a CMS; sin almacenar credenciales localmente (todo auth sigue en Stytch B2B; PostgreSQL solo guarda `stytch_member_id`/`stytch_organization_id`); sin rediseñar el contenido interno de settings/CRM/reportes/knowledge (heredan tokens, no se recomponen).

## Assumptions

- "El sitio" = presencia web completa: landing/marketing (`app/(marketing)/`) + shell del producto (`app/dashboard`). El cambio es estrictamente de presentación.
- La paleta suave propuesta (neutros cálidos + azul corporativo apagado + verde salvia) es una directriz de diseño razonable derivada de "business soft colors"; valores HSL exactos se fijan en design.md y pueden ajustarse en implementación sin cambiar el contrato de spec (contraste AA sobre blanco).
- Las utilidades `slate-900`/`emerald-500` están hardcodeadas en componentes de marketing/layout (verificado en repo); la migración es por búsqueda y reemplazo de clases hacia tokens, no reescritura de lógica.
- La eliminación de Sora por Inter-only es coherente con "leaner"; si en implementación se prefiere conservar Sora solo para el display del hero, se documenta en design.md y no altera specs.
