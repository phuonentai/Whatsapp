# Design: Consistencia visual de producto — CRM, reportes, procurement, auth, marketing y signup

## Context

- `adopt-verifika-template` dio chrome + tokens; las superficies restantes conservan clases duras residuales y semánticas de color dispares.
- El lenguaje semántico (emerald = activo/concedido, violet ✦ = IA, amber = advertencia, red = destructivo, gray = neutro; nunca color-only) ya se aplica en inbox/knowledge/dashboard; se extiende aquí.
- Sin cambios de lógica: queries, mutaciones, validaciones, contratos Stytch y endpoints intactos.

## Goals / Non-Goals

**Goals:**
- Consistencia semántica de color y lenguaje en CRM, reportes, procurement, auth/authenticate, signup, marketing.
- Estados con color + texto; destructivo red; superficies IA ✦ violet.

**Non-Goals:**
- NO reescritura de lógica/queries/acciones.
- NO cambios de rutas/SEO/metadata.
- NO nuevas funcionalidades.

## Decisions

1. **Componente de estado compartido** — reutilizar/extender el `StatusChip` (creado en `settings-redesign`) o un helper de clases de estado por semántica; un solo punto de definición de colores semánticos. Alternativa (clases inline por superficie) se descarta: ya es la causa de la dispersión.
2. **Migración por barrido** — grep de `gray-*`, `emerald-500` sueltos, `shadow-emerald-*`, `slate-900` decorativos en las rutas objetivo → reemplazo con el lenguaje/tokens. Los usos decorativos aprobados (p. ej. banner del dashboard) se conservan.
3. **Reportes** — series con `--chart-*` (tokens existentes), leyendas con texto, estado vacío honesto; sin tocar endpoints ni gating `analytics_module`.
4. **Destructivo red** — confirmar que los diálogos de eliminación usan variante destructiva (red) consistente; el flujo de confirmación no cambia.
5. **Auth/signup** — solo clases; los contratos Stytch (use-signup-flow, componentes) quedan byte a byte.

## Risks / Trade-offs

- [Barrido masivo rompe page-objects de e2e] → Mitigación: actualizar page-objects (nunca lógica); vitest de lógica pasa igual.
- [Semántica duplicada entre superficies] → Mitigación: helper/StatusChip compartido.
- [Reportes con paleta distinta por chart] → Mitigación: mapear series a `--chart-1..n` consistentemente.

## Market & Unit Economics

Este change es **visual puro**; la declaración es explícita para el council:

- **Costo: sin delta.** No se añaden llamadas LLM, ni metering, ni endpoints; solo clases/componentes compartidos en superficies existentes.
- **Funnel marketing/signup: sin cambio de lógica de conversión.** Las rutas, copy, validaciones y contratos Stytch del wizard quedan intactos; el barrido es de presentación. No se introduce ni se mide métrica de conversión nueva en este change.
- **Precios / planes: sin cambio.** No se tocan `paywall` ni `plan-pricing-ux`.
- **Métrica de negocio:** el impacto esperado es coherencia de marca (confianza B2B) — medible como follow-up de analytics si el negocio lo pide.

## Market Risk

- **R1 — Regresión de conversión por cambio visual en marketing/signup.** Un barrido de clases puede alterar jerarquía visual o estados de CTA del funnel sin intención. **Owner:** product/GTM. **Trigger:** caída de conversión o feedback cualitativo post-deploy. **Mitigación:** solo clases (sin cambios de copy/estructura); gates visuales Playwright por viewport; rollback git revert rápido.
- **R2 — Inconsistencia percibida de marca.** Si el barrido deja superficies a medio migrar, la incoherencia daña la confianza B2B. **Owner:** product/design. **Trigger:** revisión visual de QA. **Mitigación:** greps de verificación + capturas por vista en `qa/`.

## Migration Plan

1. Definir/ubicar helper de estados semánticos.
2. Migrar por superficie: CRM → reportes → procurement → auth/authenticate → signup → marketing.
3. Greps de verificación + gates.
4. Rollback: git revert.

## Open Questions

- ¿`StatusChip` de `settings-redesign` se crea aquí como dependencia o se define en este change? (dependencia: se crea en `settings-redesign`; este change lo reutiliza).
- ¿Los charts de reportes usan `--chart-*` hoy? (verificar en tasks; si usan colores fijos, se mapean a tokens).
