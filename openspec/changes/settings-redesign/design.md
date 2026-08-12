# Design: Settings redesign — lenguaje visual del diseño en los 10 módulos

## Context

- Los 10 módulos de settings existen como vistas (`?view=profile|subscription|modules|compliance|audit|whatsapp|templates|instagram|siigo|siigo-admin`) en `app/dashboard/settings/components/*` + `settings-content.tsx` (stack de vistas con overview).
- Hoy usan clases `gray-*` hardcodeadas (gris neutro sin semántica) y composición desigual entre módulos; el lenguaje del diseño (briefs UX/UI + export) exige superficies claras slate-50/white, chips semánticos emerald/amber/red con texto, tiles de color por módulo.
- Datos ya disponibles: `useAiUsageQuery` (créditos IA), `state.usage` (facturas), estados de conexión de whatsapp/instagram (queries existentes), wizard Siigo 5 pasos (`STEP_ORDER`), compliance (consent/export/anonymize).

## Goals / Non-Goals

**Goals:**
- Lenguaje visual unificado en los 10 módulos + overview.
- Uso/límites visibles en subscription (barras, umbral amber).
- Cero cambios de comportamiento/contratos.

**Non-Goals:**
- NO nuevas funcionalidades (equipo/permisos consolidado es `equipo-permisos`).
- NO cambios de rutas/URLs ni de RBAC.
- NO tocar contratos Stytch/Polar/Siigo/Meta/Instagram.

## Decisions

1. **Tokenización vs clases duras** — se migran `gray-*` → el lenguaje del diseño (slate-50/white + bordes slate-200 + emerald accents). Alternativa (reescribir a tokens shadcn `bg-card`/`border-border`) se descarta parcialmente: el diseño es explícito slate/emerald; se usa la paleta del diseño, no la indirección de tokens, para coincidir con el resto del producto (identidad Verifika). Los componentes shadcn base (`Card`, `Badge`, `Button`) se siguen usando con clases del lenguaje.
2. **Chips de estado semánticos** — un componente pequeño `StatusChip` (emerald/amber/red/gray) con icono + texto; sin color-only (a11y).
3. **Uso en subscription** — barras reutilizando `ui` copy y datos existentes; umbral amber ≥80% calculado en frontend. Sin endpoints nuevos.
4. **Estructura de archivos** — se mantienen los componentes por módulo; el cambio es clases/composición dentro de cada uno + copy en `lib/copy/ui.ts`. Sin refactor de `settings-content.tsx` salvo clases del overview.
5. **Siigo wizard** — se conserva `STEP_ORDER` y su lógica; solo clases/idiomas visuales.

## Risks / Trade-offs

- [Migración masiva de clases puede romper tests de snapshot/query de componentes] → Mitigación: vitest de settings ya cubre lógica (modules-section, whatsapp-config, siigo, subscription-tab); los selectores de lógica no cambian; actualizar solo page-objects si dependen de clases.
- [Uso de clases slate/emerald duras vs tokens de tema oscuro] → Mitigación: el lenguaje del diseño es explícito y ya es la identidad (adopt-verifika-template); tema oscuro se mantiene por tokens del shell.
- [Barras de uso sin datos] → Mitigación: estado neutro/"—", nunca fabricar porcentajes.

## Market & Unit Economics

Este change **no altera la economía unitaria**: es un re-estilo visual de los 10 módulos de settings sobre datos ya existentes.

- **Costo de IA: sin delta.** No se añaden llamadas LLM, ni metering, ni routing nuevo. `useAiUsageQuery` y `state.usage` ya se consumen hoy; solo se visualizan mejor (barras con umbral).
- **Precios / planes / créditos: sin cambio.** No se tocan `paywall`, `plan-pricing-ux` ni `billing-quota-integrity`; la lógica de cancelación/resumen/cambio de plan en `subscription-tab.tsx` queda intacta (solo clases/copy). Las ramas de verificación de pago Polar/MercadoPago no están en este change (viven en `app/dashboard/page.tsx`).
- **Margen por plan: sin impacto.** No hay flujo de cobro, fee ni descuento modificado.
- **Conversión / retención:** la única expectativa es presentacional (mejor comprensión de uso y límites puede reducir churn por sorpresa de límites); no se introduce ni se mide una métrica nueva en este change; si el negocio quiere medirlo, es follow-up de analytics.

## Market Risk

- **R1 — Percepción de límites (billing).** Mostrar barras de uso con umbral amber puede alertar a clientes sobre límites que antes ignoraban, generando preguntas de soporte o presión de upgrade. **Owner:** product/GTM. **Trigger:** aumento de tickets sobre límites o feedback de clientes. **Mitigación:** umbral amber con copy clara de "ver plan"; los límites ya existen (no se cambian); el estado neutro/"—" sin datos evita alarmas falsas.
- **R2 — Copy de chips de estado (WhatsApp/Meta channel adjacency).** Los chips "conectado/pausado" no deben afirmar estados de la plataforma Meta que el backend no confirme (regla de honestidad). **Owner:** product/ops. **Trigger:** cambio de términos de Meta o copy que afirme estado sin confirmación. **Mitigación:** chips alimentados de las queries de conexión existentes, nunca hardcodeados.

## Migration Plan

1. Crear `StatusChip` + utilidades de lenguaje (si aplica) en `components/ui/` o `components/layout/`.
2. Migrar módulo por módulo (profile → subscription → modules → compliance → audit → whatsapp → templates → instagram → siigo → siigo-admin → overview), con copy en `ui.ts`.
3. Gates: lint/build/tsc + vitest de settings + visual Playwright.
4. Rollback: git revert; sin estado de DB/Stytch.

## Open Questions

- ¿Los page-objects de e2e existentes dependen de clases `gray-*`? (se verifica en tasks; actualizar page-object, no lógica).
- ¿El umbral amber debe configurarse por plan o es fijo 80%? (v1: fijo 80%, se documenta).
