# Proposal: Consistencia visual de producto — CRM, reportes, procurement, auth, marketing y signup

## Why

El lenguaje visual del diseño (briefs UX/UI + export) define semánticas de color que hoy no están aplicadas de forma consistente en las superficies restantes: **emerald = concedido/activo, violet ✦ = IA, amber = advertencia/escalación, red = destructivo/error, gray = neutro** — y la regla de nunca color-only. `adopt-verifika-template` ya dio el chrome y los tokens, pero CRM, reportes, procurement, auth, marketing y signup mantienen clases duras residuales (`gray-*`, emerald sueltos, estados sin texto) y semánticas dispares. Este change cierra la consistencia visual de esas superficies sin tocar lógica, rutas ni contratos.

## What Changes

- **CRM** (`app/dashboard/crm/*`): tarjetas/tablas/etiquetas al lenguaje — badges de estado (etapa de negocio, etiquetas) con color + texto; acciones destructivas (eliminar) en red consistente; botones primarios emerald; sin cambio de queries/acciones.
- **Reportes / analytics** (`app/dashboard/reportes/*`): charts y tablas al lenguaje — paleta de series consistente con tokens (`--chart-*`), leyendas con texto, estado vacío honesto (sin fabricar), tablas con `th scope="col"`; sin cambios de endpoints (usa analytics existentes).
- **Procurement + inquiry schedules** (`app/dashboard/procurement/*`): tarjetas/estados (proveedores, cronogramas) al lenguaje — chips de estado con texto; sin cambio de flujo.
- **Auth / authenticate / passwordless** (`app/auth/*`, `app/authenticate/*`): superficies al lenguaje (mismo chrome/inputs que sign-in); contratos Stytch intactos.
- **Signup wizard** (`app/signup/*`): pasos visuales DIAN (NIT, régimen, ciudad) al lenguaje; `use-signup-flow.ts` y componentes Stytch intactos.
- **Marketing** (`app/(marketing)/*`, `components/marketing/*`): barrido de clases duras residuales (`emerald-500` sueltos, `slate-900` decorativos no aprobados, sombras emerald) → tokens/lenguaje; rutas, metadata, SEO sin cambios.
- **Reglas semánticas**: componente compartido de estado (chip color+texto) reutilizado; botones destructivos red; superficies IA con ✦ violet donde aplique; nada color-only.

## Capabilities

### New Capabilities

- (ninguna — consistencia sobre capacidades existentes)

### Modified Capabilities

- `crm-frontend`: las superficies CRM adoptan el lenguaje visual semántico (estados con color+texto, destructivo red) sin cambios de comportamiento.
- `analytics`: reportes adoptan la paleta de tokens y el lenguaje visual; sin cambios de endpoints.
- `inquiry-scheduling`: procurement/cronogramas adoptan el lenguaje visual.
- `signup-stytch-compliance`: el wizard adopta el lenguaje visual (pasos DIAN visuales ya presentes) sin cambios de contrato Stytch.
- `marketing-website`: barrido final de clases duras residuales al lenguaje; rutas/SEO intactos.

## Impact

- **Frontend only**: `next_b2b_starter/app/dashboard/crm/*`, `app/dashboard/reportes/*`, `app/dashboard/procurement/*`, `app/auth/*`, `app/authenticate/*`, `app/signup/*`, `app/(marketing)/*`, `components/marketing/*`, `lib/copy/ui.ts`. Sin backend, sin migraciones, sin SQLC, sin cambios de API.
- **Auth**: cero cambios de contrato Stytch (solo clases).
- **Billing**: cero cambios (no toca ramas de pago).
- **Dependencias**: ninguna nueva.
- **Ops**: `pnpm build`, `pnpm lint`, `npx tsc --noEmit`; vitest existente de CRM/reportes/procurement/signup; Playwright visual/a11y → `qa/`.
- **Rollback**: git revert; sin estado de DB ni de Stytch.
- **Non-Goals**: sin reescritura de lógica/queries/acciones; sin cambios de rutas/SEO/metadata; sin nuevas funcionalidades; sin almacenar credenciales localmente (todo auth sigue en Stytch B2B).

## Assumptions

- Las superficies ya tienen el chrome del template (adopt-verifika-template); este change es el barrido de consistencia semántica y clases duras residuales.
- La semántica de color (emerald/amber/red/violet) ya se aplica en inbox/knowledge/dashboard; se extiende a CRM/reportes/procurement/auth/marketing/signup.
- Los tests existentes cubren lógica; si algún page-object depende de clases de estilo, se actualiza el page-object (no la lógica).
