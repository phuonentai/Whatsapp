# Proposal: Admin panel — superficie de operador (orgs, uso IA, auditoría cross-org)

## Why

El inventario de páginas lista las superficies de plataforma/operador: shell de admin con vistas cross-org (orgs, uso, auditoría) y vista de uso/créditos IA por org (tablas por org, tablas de tasa de modelo). Hoy solo existe la superficie Siigo del tenant (`siigo-admin-view`, onboarding/provisioning en settings, org-scoped) que queda FUERA del alcance de la plataforma; no hay un shell de operador. Las specs `admin-panel-navigation`/`admin-panel-audit-log` y `ai-usage-metering` ya existen (ledger de uso IA persistido y audit de org), por lo que la vista de uso y la auditoría tienen fuente de datos. Este change construye el shell de operador y sus vistas, con gate de plataforma (rol de operador en Stytch o entitlement), sobre la UI del lenguaje del diseño.

## What Changes

- **Shell de operador** (`/admin` o ruta separada, fuera del layout de `/dashboard`): sidebar de plataforma (Organizaciones, Uso IA, Auditoría) con el lenguaje del diseño; gate de acceso por rol de operador (permiso `platform:operate` o rol `platform_admin` en Stytch) — no accesible para miembros normales.
- **Siigo FUERA de alcance de la plataforma**: ni sección, ni credenciales, ni estado de conexión, ni datos de facturación. `siigo-admin-view` permanece en settings del tenant (org-scoped, gate rol `admin`, sin cambios). Dos paneles con frontera estricta: el panel del tenant ve sus datos (incl. Siigo); la plataforma solo estado operativo cross-org.
- **Vista Organizaciones (cross-org)**: tabla de orgs (nombre, membresía, plan/estado de suscripción, estado de conexión WhatsApp/Instagram — NO Siigo) con búsqueda y paginación; detalle de org con estado de conexiones e integraciones (sin Siigo, sin CRM/contactos).
- **Vista Uso IA / créditos (plataforma)**: tabla por org del periodo (tokens input/output/embedding, créditos usados, límite `ai_credits_max`, % de uso) desde el ledger `ai_usage`; tablas de tasa de modelo (modelo, precio/token, feature) como referencia; filtros por periodo y org.
- **Vista Auditoría cross-org**: eventos del ledger (`ai_usage_events` append-only) y actividad operativa de plataforma (estado de conexiones/suscripción) por org con filtros (org, tipo, fecha); read-only y acotada a datos operativos — sin actividad CRM, contactos ni contenido de conversaciones en v1 (purpose limitation Ley 1581).
- **Siigo admin**: FUERA de alcance — la vista permanece en settings del tenant sin cambios; la plataforma no la expone.
- **Gates y a11y**: acceso solo con permiso de plataforma (403 sin él); tablas reales con `th scope="col"`, estados con texto (nunca color-only), skeleton/error/empty states.

## Capabilities

### New Capabilities

- `admin-panel-surface`: shell de operador con vistas cross-org (organizaciones, uso IA, auditoría), gate de plataforma, sobre el ledger `ai-usage-metering` y la auditoría existentes; Siigo queda fuera de alcance de la plataforma.

### Modified Capabilities

- `admin-panel-navigation`: la navegación de plataforma pasa de un sidebar parcial (inbox/CRM) a un shell de operador con secciones Organizaciones / Uso IA / Auditoría; la navegación de `/dashboard` no cambia.
- `admin-panel-audit-log`: la auditoría gana una vista cross-org de plataforma (ledger y actividad por org) además de la vista por-org existente en settings.
- `ai-usage-metering`: se añade la superficie de lectura de plataforma (tabla por org + tablas de tasa de modelo) sobre el ledger existente; sin cambios al modelo de datos.
- `settings-ui`: SIN CAMBIOS — la vista `siigo-admin` permanece en settings del tenant tal cual (coordinación: `settings-redesign` la re-estiliza sin cambiar su gate).

## Impact

- **Backend**: `go-b2b-starter/` — grupo de rutas de plataforma `/api/v1/platform/*` read-only (lista de orgs con estado de suscripción/conexiones, uso IA agregado por org/periodo, tasas de modelo, auditoría cross-org) con middleware de contexto de plataforma (sin scoping de tenant, validación de `org_id` contra `organizations`, fallback 403/503) y gate `platform:operate`; una migración aditiva `platform_access_log` (append-only, sin ALTERs sobre tablas existentes, retención 90 días) para auditoría de acceso; sin otras migraciones nuevas (usa `ai_usage`, `ai_usage_events`, `quota_tracking`, orgs existentes). Tests de 403/200 por rol y de no-fuga cross-org.
- **Frontend**: `next_b2b_starter/app/admin/*` (shell + vistas) o ruta dedicada; `components/layout/` (sidebar de plataforma); `lib/copy/ui.ts`.
- **Auth**: nuevo permiso `platform:operate` en Stytch (runtime SSOT) sobre una org de plataforma dedicada (`platform-ops`) con rol `platform_admin` — las sesiones B2B son siempre org-scoped (docs Stytch), por lo que el operador es miembro de esa org; rollback = revert Git + ajuste de política Stytch documentado.
- **Dependencias**: ninguna nueva.
- **Ops**: `make test` (Go), `pnpm build`/`lint`/`tsc`, vitest, Playwright visual/a11y → `qa/`.
- **Rollback**: git revert; sin estado de negocio en DB (solo lectura; `platform_access_log` es audit-only y sin ALTERs); política Stytch revertible.
- **Non-Goals**: NO gestión cross-org de datos de negocio (solo lectura/estado); NO exposición cross-org de datos de CRM/contactos/conversaciones en v1; NO exposición de Siigo en la plataforma (credenciales, estado, facturación — permanece en settings del tenant); NO facturación cross-org (se ve estado, no se cobra); NO edición de orgs en v1; NO migraciones sobre tablas existentes (solo tabla aditiva de acceso); NO almacenar credenciales localmente (todo auth en Stytch B2B).

## Assumptions

- El ledger `ai_usage`/`ai_usage_events` y `quota_tracking.ai_credits_max` existen (spec `ai-usage-metering` vigente) y son consultables agregados por org/periodo.
- El rol de operador de plataforma se introduce en la política Stytch con un permiso `platform:operate`; los miembros normales no lo tienen.
- Los operadores son miembros de una org de plataforma dedicada (`platform-ops`) con rol `platform_admin` — grounded en docs Stytch (las sesiones B2B son siempre org-scoped, `authorization_check` acopla a la org de la sesión; la autorización cross-org se resuelve con política cacheada + claim `roles` del JWT). Si Stytch expone un modelo de miembro de plataforma sin org, se valida en el spike y se ajusta (assumption a validar).
- `siigo-admin-view` permanece en settings del tenant (org-scoped, gate rol `admin`, sin cambios); la plataforma no expone dato alguno de Siigo.
- La tabla de tasas de modelo puede derivarse de la configuración de metering existente o de una tabla de referencia nueva de solo lectura (spike en tasks).
