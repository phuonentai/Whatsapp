# Design: Rediseño del sitio — ligero + colores empresariales suaves

## Context

NexoChat es un CRM para WhatsApp B2B (mercado Colombia/LATAM) con frontend Next.js 16 en `next_b2b_starter/`. El sitio actual sigue el template Shuffle aprobado (dark `slate-900` + `emerald-500` saturado + Sora): secciones oscuras pesadas en la landing (`app/(marketing)/`), shell fijo oscuro en el producto (`components/layout/*`) y tokens shadcn neutros (hue 240, primario casi negro). El equipo UI/UX/SEO/diseño lo considera pesado y pide un rediseño **más ligero** con **colores empresariales suaves**.

Estado verificado en repo:
- Marketing: hero `bg-slate-900` con gradientes emerald, botones `bg-emerald-500` + `shadow-emerald-500/25`, header/footer oscuros, `Reveal` (framer-motion) en secciones, Sora en `components/marketing/fonts.ts`.
- Shell: `sidebar.tsx`/`header.tsx` con `bg-slate-900`, `border-slate-800`, acentos `text-emerald-400`, tarjeta "IA Insights" `from-emerald-500/20`.
- Producto: overview con verificación de pagos (Polar `checkout_id` → `verifyPayment`; MercadoPago `payment_id`/`preapproval_id` → `verifyMercadoPagoPayment`) **no debe tocarse**; bandeja y signup re-estilizables a nivel de clases.
- Specs que gobiernan el shell: `dashboard-template-restyle` (requiere shell oscuro slate-900) y `app-shell` (requiere shell fijo oscuro en ambos temas) — ambas se modifican vía delta specs en este change. El sitio de marketing no tiene spec propia; se introduce `marketing-site-visual`.

Restricciones: cambio estrictamente de presentación; sin tocar auth Stytch, RBAC, billing, datos ni rutas. Fuentes de truth: `openspec/specs/` (vivos) y los deltas de este change.

## Goals / Non-Goals

**Goals**
1. Paleta empresarial suave coherente (neutros cálidos, azul corporativo apagado como primario, verde salvia como acento secundario de marca) aplicada como tokens de tema reutilizables.
2. Sitio público más ligero: menos peso visual por sección, hero sin fondos oscuros ni gradientes, motion mínimo, tipografía Inter-only (sin Sora).
3. Shell del producto suave (claro y oscuro) coherente con la marca, sin cambiar navegación/RBAC/permisos.
4. Cero cambios de comportamiento: verificación de pagos, flujo Stytch, filtros de bandeja, wizard de signup y KPIs ("—" sin fuente) intactos.

**Non-Goals**
- No cambiar rutas, metadata, sitemap, robots ni structured data (SEO-estructura intacta; solo mejora de LCP por menos animación).
- No reescribir copy ni contenido (solo se recorta presentación).
- No nuevos features ni componentes nuevos de negocio.
- No tocar backend, DB, auth, RBAC ni billing.
- No eliminar el soporte de tema oscuro (variante oscura suave).
- No rediseñar el contenido interno de settings/CRM/reportes/knowledge (heredan tokens; no se recomponen).
- No almacenar credenciales localmente (todo auth sigue en Stytch B2B; PostgreSQL solo guarda `stytch_member_id`/`stytch_organization_id`).

## Decisions

### D1. Tokens de tema suaves en `app/globals.css` (ambos temas)

Sustituir la paleta shadcn neutra (hue 240, primario negro) por una paleta empresarial suave. Valores propuestos (HSL, sujetos a ajuste fino en implementación manteniendo contraste AA):

| Token | Light | Dark (suave, no negro puro) |
|---|---|---|
| `--background` | `60 33% 99%` (blanco cálido) | `222 15% 10%` |
| `--foreground` | `222 18% 22%` (carbón suave) | `210 20% 92%` |
| `--card` | `0 0% 100%` | `222 13% 13%` |
| `--primary` | azul corporativo apagado (`≈217 42% 46%`, ver D4) | `217 55% 62%` (azul más claro) |
| `--primary-foreground` | `0 0% 100%` | `222 15% 10%` |
| `--secondary` | verde salvia `150 28% 93%` | `150 20% 20%` |
| `--secondary-foreground` | `150 24% 26%` | `150 25% 80%` |
| `--muted` | `60 12% 96%` | `222 12% 16%` |
| `--muted-foreground` | `222 12% 45%` | `215 15% 65%` |
| `--border` | `220 12% 90%` | `222 12% 22%` |
| `--ring` | `217 45% 55%` | `217 55% 60%` |

Racional: hue 240 (púrpura frío) → hue 220/217 (azul) transmite confianza B2B; neutros con saturación baja (12–33%) evitan el look "frío"; el verde salvia (hue 150, saturación baja) conserva la asociación WhatsApp/Siigo sin el emerald saturado. Alternativa considerada: mantener emerald como primario (descartada: es el acento que el equipo critica como poco empresarial); azul marino oscuro como primario (descartada: demasiado pesado, contradice "soft").

### D2. Tipografía Inter-only (retirar Sora)

`components/marketing/fonts.ts` deja de exportar `sora` y reutiliza la variable Inter ya cargada en `app/layout.tsx`. El alias `font-heading` en `tailwind.config.ts` se mantiene pero apunta a la variable Inter (`--font-heading` = variable de Inter), de modo que los usos de `font-heading` en componentes no cambian. Se elimina la importación de `Sora` de `next/font/google` (menos descarga de fuentes → mejor LCP/CLS).

Racional: Sora es una display font "tech/startup" que añade peso visual; Inter-only es el estándar B2B suave (Stripe, Linear, Notion usan variantes de una sola familia). Alternativa: conservar Sora solo en el hero (descartada por simplicidad y por la directiva "leaner"; se documenta como open question resuelta por defecto).

### D3. Composición marketing ligera (mismas secciones, menos peso)

Sobre `components/marketing/*` y `app/(marketing)/*` sin cambiar rutas ni copy:
- Hero: `bg-slate-900` + gradientes emerald → superficie clara (`bg-background`/`bg-card` con borde sutil); badge estático (sin `animate-pulse`); h1 de `text-6xl` a `text-5xl`; sombras grandes eliminadas.
- Secciones: `py-20 lg:py-32` → `py-16 lg:py-24`; alternancia de superficies `bg-white`/`bg-muted/50` con `border-y` sutiles en vez de bloques oscuros.
- Botones primarios: `bg-emerald-500` → `bg-primary` (azul suave) con hover `bg-primary/90`; sin `hover:scale-105` (el scale es el anti-patrón "pesado").
- `Reveal`/framer-motion: fade-up 300ms sutil; desactivado por defecto en el hero (LCP) y en `prefers-reduced-motion`; sin scroll-jacking.
- ROI calculator, comparison, feature grid, pricing, FAQ, CTA, footer: mismos componentes, misma lógica (sliders, toggle, acordeón), solo clase/estilo.
- Footer oscuro 4-col → claro con bordes suaves.

Racional: "leaner" se implementa reduciendo capas decorativas (gradientes, sombras, animaciones, padding heroico) en lugar de eliminar secciones/información — mantiene el valor SEO/conversión de la landing y reduce coste de pintura. Alternativa: cortar secciones (descartada: perdería mensajes de venta validados; el equipo pidió "leaner", no "shorter").

### D4. Contraste AA como invariante

El azul primario se fija en implementación dentro de `217 40–48% L`, con verificación de contraste: ≥4.5:1 texto normal sobre blanco/card, ≥3:1 texto grande/UI. Si un tono falla AA, se ajusta la luminosidad (nunca la saturación por encima de ~50%) antes de aprobar. Los acentos decorativos (salvia) nunca se usan para texto de cuerpo.

Racional: paletas "soft" suelen fallar AA; el requisito de contraste es un contrato de accesibilidad del rediseño y será verificado por el stage `/uiux` (Playwright a11y) y manualmente.

### D5. Shell del producto suave (claro y oscuro)

`components/layout/*`:
- Sidebar: `bg-slate-900` → `bg-card`/`bg-background` con `border-r border-border`; ítem activo `bg-primary/10 text-primary`, hover `bg-muted`; grupos "Inteligencia Artificial" y tarjeta "IA Insights" conservan su render condicional RBAC, re-estilizada con tintes salvia suaves (`bg-secondary/60 border-secondary`).
- Header: `bg-slate-900/95` → `bg-background/95 backdrop-blur` con `border-b border-border`; search (⌘K), notificaciones y user-menu sin cambios de comportamiento.
- En tema oscuro el shell usa las superficies oscuras suaves del token (D1), coherente con el contenido.

Racional: el shell oscuro fijo era un artefacto del template aprobado; la nueva directriz de marca lo reemplaza por superficies de tema, eliminando la regla "shell siempre oscuro" que además rompía con los tokens. Alternativa: mantener shell oscuro y solo suavizar contenido (descartada: la directiva es explícita sobre colores suaves; y `app-shell` ya exigía el shell oscuro solo por fidelidad al template, hoy superada).

### D6. Preservación de comportamiento como contrato

- `app/dashboard/page.tsx`: las ramas `verifyPayment`/`verifyMercadoPagoPayment` y sus redirects NO se tocan (solo clases de las tarjetas KPI/paneles alrededor).
- Bandeja: filtros de canal/estado, queries y store intactos (solo clases).
- Signup: `use-signup-flow.ts` y componentes Stytch intactos; el contenedor del wizard pasa a superficie suave (el fondo oscuro del template se sustituye por `bg-background`); Stytch embedded conserva su propio theming.
- e2e Playwright: si algún page-object depende de clases de estilo (p.ej. `bg-slate-900`), se actualiza el page-object; los selectores de lógica (`data-testid`/roles) no cambian.

## Risks / Trade-offs

- [Contraste AA insuficiente en azul suave] → Mitigación: rango de luminosidad documentado (D4), verificación manual + stage `/uiux`; botones usan `--primary-foreground` blanco.
- [E2E depende de clases de estilo] → Mitigación: barrido de page-objects; mantener atributos semánticos; ejecutar suite completa en el gate.
- [Pérdida de identidad de marca (emerald/WhatsApp)] → Mitigación: verde salvia como acento secundario + glifo WhatsApp y logos Siigo/MP/Meta preservados; revisión visual por el equipo.
- [Dark mode regresión (specs `app-shell`/`dashboard-template-restyle`)] → Mitigación: delta specs de este change actualizan las requirements; ambos temas se verifican en QA.
- [Sora→Inter altera métricas de layout (CLS/visual)] → Mitigación: `next/font` swap ya presente; QA visual en 390/768/1440.
- [Scope creep hacia reescritura de contenido] → Mitigación: cambios de copy permitidos solo si un componente se recorta; cualquier cambio de contenido se registra como tarea explícita y se revisa con el equipo.
- [Regresión en flujos de pago por tocar `app/dashboard/page.tsx`] → Mitigación: regla D6 — diff del gate debe mostrar cero cambios en ramas de verificación.

## Migration Plan

1. **Fase A — Tokens**: `app/globals.css` (paleta D1), `tailwind.config.ts` (alias `font-heading` → Inter), `components/marketing/fonts.ts` (retirar Sora), verificación `pnpm build`.
2. **Fase B — Marketing**: barrido `grep -rn "slate-900\|emerald-" components/marketing/ app/(marketing)/` → reemplazo por tokens (D3); motion reducido (D3); build + lint.
3. **Fase C — Shell y producto**: `components/layout/*` (D5); clases en `app/dashboard/page.tsx` (solo clases, D6), `app/dashboard/inbox/*`, `app/signup/*`; build + lint.
4. **Fase D — Verificación (gate)**: `pnpm build`, `pnpm lint`, `pnpm dev` + smoke; e2e Playwright existentes (dashboard/inbox/signup/CRM/marketing); verificación manual de retorno de checkout Polar y MercadoPago (payment_verified/error); chequeo de contraste AA (D4); stage `/uiux` (Playwright a11y + visual en 390x844/768x1024/1440x900 → `qa/`); actualizar page-objects si aplica.
5. **Rollback**: `git revert` del change completo (o revert por fases A→C). No hay estado de DB ni de Stytch que revertir. Los tokens previos y la fuente Sora se recuperan del historial.

## Open Questions

- **Azul primario exacto**: se fija en implementación dentro del rango D4; se requiere validación visual del equipo antes de cerrar el change (se deja registrado en tasks como paso de verificación).
- **Sora en hero**: por defecto NO (Inter-only). Si el equipo quiere un acento display mínimo, se añade como tarea acotada sin cambiar specs.
- **Wizard de signup**: por defecto pasa a superficie suave clara (consistencia con D5); el embedded de Stytch conserva su propio tema — confirmar en QA visual.
- **Footer**: por defecto claro con bordes suaves; confirmar en QA visual.
