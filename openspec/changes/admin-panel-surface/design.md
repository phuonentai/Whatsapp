# Design: Admin panel — superficie de operador (orgs, uso IA, auditoría cross-org)

## Context

- Inventario: shell de operador + cross-org (orgs, uso, auditoría) + vista de uso/créditos IA por org (tablas por org, tablas de tasa de modelo).
- Specs existentes: `ai-usage-metering` (ledger `ai_usage`, `ai_usage_events`, `quota_tracking.ai_credits_max`), `admin-panel-audit-log` (vista por-org en settings), `admin-panel-navigation`.
- Hoy no hay shell de operador; `siigo-admin-view` es la superficie Siigo del TENANT (onboarding/provisioning dentro de settings, org-scoped) y queda FUERA del alcance de la plataforma.
- Clean Architecture + Stytch runtime SSOT aplican igual: el rol de plataforma vive en la política Stytch.
- No existe hoy ninguna superficie cross-org backend: cada request resuelve un único `OrganizationID` en el seam `authcontext` (`GetOrganizationID`); los endpoints Siigo son org-scoped (`/v1/org/n/...`). Las rutas de plataforma necesitan un contexto de plataforma explícito (ver Decisión 7).

## Goals / Non-Goals

**Goals:**
- Shell de operador con gate `platform:operate`.
- Vistas read-only: organizaciones (estado), uso IA por org (ledger), auditoría cross-org.
- Lenguaje visual del diseño en toda la superficie.

**Non-Goals:**
- NO gestión cross-org de datos de negocio (solo lectura).
- NO exposición cross-org de datos de CRM/contactos/conversaciones en v1 — las vistas de plataforma muestran SOLO estado operativo y uso IA (purpose limitation Ley 1581).
- NO exposición de Siigo en la plataforma: ni credenciales, ni estado de conexión, ni datos de facturación/numeración — Siigo permanece en settings del tenant (org-scoped, sin cambios).
- NO facturación cross-org.
- NO edición de orgs en v1.
- NO migraciones sobre tablas existentes: a lo sumo una tabla aditiva de auditoría de acceso (sin ALTERs, sin locks de escritura).

## Decisions

1. **Ruta `/admin` separada del layout de `/dashboard`** — el shell de operador tiene su propio layout/sidebar; los miembros normales no ven enlaces y reciben 403 directo. Alternativa (dentro de `/dashboard`) se descarta: confunde audiencias y el gate sería por vista en vez de por shell.
2. **Permiso `platform:operate` en Stytch** — rol de plataforma; el enforcement server-side lo exige en todos los endpoints del shell vía el servicio de política RBAC existente (`stytch-authorization`, cache Redis 5 min, circuit breaker de dos niveles). **Modelo de operador (grounded en docs Stytch):** las sesiones B2B SIEMPRE están scoped a una sola organización (`organization_id` en el JWT, inmutable por sesión — stytch.com/docs, sessions/overview), por lo que no existe una sesión "sin org". Los operadores de plataforma SHALL ser miembros de una organización dedicada de plataforma (org `platform-ops`, slug reservado) con rol `platform_admin` asignado por `direct_assignment`, que otorga `platform:operate` (recurso `platform`, acción `operate` — patrón `resource:action` documentado en RBAC overview). Contrato de fallback: **403** si el miembro no tiene el permiso; **503** si la política no está disponible y la caché está vacía (nunca un falso allow ni un falso "sin permisos"). Rollback documentado (revert Git + ajuste de política).
   - **NOTA (`authorization_check`):** el check de autorización de Stytch (`sessions/authenticate` con `authorization_check`) verifica que el `organization_id` del check coincida con la org de la sesión (403 si difiere — enforcing-permissions). Por diseño, NO se usa `authorization_check` para autorizar lecturas cross-org: la autorización de plataforma se resuelve con la política cachead (Get Policy) + los roles del miembro (claim `roles` del JWT), y la existencia del `org_id` objetivo se valida aparte contra `organizations`. Un operador en `platform-ops` NO pasa `authorization_check` con orgs ajenas.
   - **Gating en edge (opcional, best practice):** custom claim template `{{ member.rbac.platform.actions }}` en el JWT permite gatear el shell en edge sin llamada API por request; se valida localmente contra el JWT ya verificado por JWKS (cache TTL <= 300s, invariante existente).
3. **Endpoints read-only de plataforma** — lista de orgs con estado (suscripción vía `subscription`/Polar o estado de conexiones), agregado de uso IA por org/periodo (consulta sobre `ai_usage`), eventos de auditoría filtrables, tasas de modelo (tabla de referencia read-only o derivada de configuración). Sin escrituras.
4. **Uso IA** — agregación en SQL sobre el ledger (sin nueva tabla); % de uso vs `ai_credits_max`; filtros periodo/org; sin datos → 0/"—". Las agregaciones platform-wide (todas las orgs de un periodo) SHALL validarse en el spike 1.1: forma de query e cobertura de índices (los índices actuales son org-first: `idx_ai_usage_org_period`, `idx_ai_usage_events_org_created`); paginación límite obligatoria server-side; si se requiere un índice period-first, se introduce como migración aditiva separada (expand-contract), fuera de este cambio si no es estrictamente necesario.
5. **Siigo FUERA de alcance de la plataforma** — el shell de operador NO incluye sección Siigo ni dato alguno de Siigo (estado, credenciales, facturación, numeración). `siigo-admin-view` permanece en settings del tenant con su gate de rol `admin` y endpoints org-scoped existentes (`/v1/org/siigo/...`), sin cambios. Decisión de producto: dos paneles con frontera estricta — el panel del tenant ve sus datos (incl. Siigo); el panel de plataforma ve solo estado operativo cross-org (orgs, uso IA, auditoría). Esto elimina el conflicto con `settings-redesign` (que re-estiliza `siigo-admin` en settings) y con el living spec `admin-panel-navigation` (onboarding overview de Siigo gateado al rol `admin` del tenant).
6. **Componentes** — tablas reales (`th scope="col"`), estados con texto, skeleton/error/empty; copy en `lib/copy/ui.ts`.
7. **Contexto de plataforma (request-context model)** — las rutas de plataforma forman un grupo explícito `/api/v1/platform/*` donde el middleware de scoping por tenant se REEMPLAZA por un middleware de contexto de plataforma: autentica la sesión Stytch existente (JWT verificado por JWKS en edge / `X-Forwarded-Auth`), resuelve el miembro (claims `member_id` + `organization_id` del JWT) y coloca en el contexto un principal DE PLATAFORMA (miembro + org `platform-ops` como identidad; SIN tenant de datos). El permiso `platform:operate` se resuelve desde el servicio de política cacheado + roles del JWT (sin llamada Stytch por request; fallback 403/503 según Decisión 2). El filtro `org_id` es un parámetro de consulta VALIDADO contra `organizations` (400 si malformado, 404 si no existe), nunca un mecanismo de scoping derivado del llamante: el scoping de datos deriva del principal de plataforma + filtro validado. Los endpoints member-scoped NO sirven datos cross-org y las rutas de plataforma NO heredan el scoping del contexto de miembro. Pruebas por rol: 200/403 en rutas de plataforma; regresión de que un operador NO puede leer datos de otra org vía endpoints org-scoped (scoping intacto) y que un miembro sin permiso recibe 403.
8. **Límite de datos cross-org (v1)** — las vistas de plataforma exponen SOLO: identidad de org (nombre, conteo de miembros), estado de suscripción, estado de conexión WhatsApp/Instagram (NO Siigo), agregados del ledger `ai_usage` y eventos `ai_usage_events`. Actividad CRM (notas, llamadas, correos, mensajes WhatsApp), contactos, contenido de conversaciones y TODO lo de Siigo (credenciales, estado, facturación) SHALL quedar excluidos de la plataforma en v1 (ver deltas `admin-panel-audit-log` y `admin-panel-surface`).
9. **Auditoría de acceso de plataforma** — toda lectura cross-org (listado, búsqueda, filtro, detalle) SHALL registrarse en una tabla aditiva append-only `platform_access_log` (nueva migración, sin ALTERs): `actor_stytch_member_id`, `actor_stytch_organization_id`, `target_organization_id` (nullable), `action`, `created_at`. Retención 90 días (configurable). Las superficies read-only nunca mutan datos de negocio.

## Risks / Trade-offs

- [Agregación sobre ledger grande] → Mitigación: índices existentes (validar cobertura period-first en spike 1.1) + límites de paginación obligatorios server-side; consulta agregada por org/periodo. Si el spike detecta falta de índice period-first para agregación platform-wide, se documenta como decisión expand-contract en tareas (migración aditiva separada, nunca dentro de la superficie read-only).
- [Estado de suscripción de org no agregado hoy] → Mitigación: consulta sobre la tabla de suscripción existente o el status endpoint; si no hay fuente única, se muestra lo disponible + "—".
- [Rol de plataforma nuevo en Stytch] → Mitigación: gate server-side + tests 403; rollback documentado. Modelo grounded en docs Stytch (sesiones org-scoped → org dedicada `platform-ops`); validar en spike 1.1 si Stytch expone alternativa de miembro sin org y si el slug/org dedicada es viable en el tenant Stytch.
- [Siigo fuera de alcance de la plataforma] → Sin riesgo de e2e: `siigo-admin-view` permanece en settings del tenant sin cambios; ninguna ruta de plataforma toca tablas o campos Siigo. Regresión: tests que verifican ausencia de campos Siigo en respuestas de plataforma.

## Market & Unit Economics

Este change **expone datos de metering existentes a la operación**; no altera la economía:

- **Costo de IA: sin delta.** No se añaden llamadas LLM ni metering nuevo; la vista Uso IA agrega el ledger `ai_usage` existente (solo lectura).
- **Precios / planes / créditos: sin cambio.** No se tocan `paywall`, `plan-pricing-ux` ni `ai-usage-metering`; la tabla de tasas de modelo es referencia read-only.
- **Margen / cobro: sin impacto.** La plataforma NO factura ni cobra cross-org; solo lee estado.
- **Métrica de negocio:** la vista Uso IA habilita el seguimiento operativo del consumo por org (follow-up natural: alertas de uso); no se introduce métrica nueva en este change.

## Market Risk

- **R1 — Exposición de datos cross-org.** La plataforma agrega datos de múltiples orgs (uso, auditoría, estado de conexión); fuga = riesgo de cumplimiento (Ley 1581 adjacency) y confianza. **Owner:** security. **Trigger:** incidente de acceso o auditoría. **Mitigación (concreta):** gate `platform:operate` server-side con contrato 403/503 (Decisión 2), vistas read-only, alcance de datos acotado a estado operativo + uso (Decisión 8, excluye CRM/contactos/conversaciones), y auditoría de acceso de plataforma en `platform_access_log` con retención 90 días (Decisión 9).
- **R2 — Expectativa de gestión desde la plataforma.** Vistas read-only pueden generar demanda de "arreglar desde plataforma" (editar org, pausar suscripción). **Owner:** product. **Trigger:** solicitudes de operadores. **Mitigación:** scope explícito read-only en v1; follow-up separado si el negocio lo pide.
- **R3 — Costo operativo de la tabla de tasas de modelo.** Si la tabla de referencia se desactualiza, el negocio puede tomar decisiones de pricing con datos viejos. **Owner:** product/backend. **Trigger:** cambio de modelos o precios. **Mitigación:** tabla de referencia marcada como tal; fuente derivada de la configuración de metering (spike en tasks 1.1).

## Migration Plan

1. Política Stytch: añadir rol/permit `platform:operate` (solo operadores).
2. Backend: middleware de contexto de plataforma (grupo `/api/v1/platform/*`, sin scoping de tenant, validación de `org_id`, fallback 403/503) + endpoints read-only de plataforma + tests por rol (200/403 + regresión de no-fuga cross-org).
3. Backend: migración aditiva `platform_access_log` (append-only, retención 90 días) + registro de acceso en toda lectura cross-org.
4. Frontend: shell `/admin` + vistas (orgs, uso IA, auditoría) + navegación. Sin sección Siigo (permanece en settings del tenant).
5. Gates: `make test`, lint/build/tsc, vitest, Playwright visual/a11y.
6. Rollback: revert Git + ajuste de política Stytch (sin estado de negocio afectado; `platform_access_log` es audit-only).

## Open Questions

- ¿Existe una fuente única para el estado de suscripción de cada org en la plataforma? (spike en tasks 1.1).
- ¿Las tasas de modelo se derivan de la configuración de metering o requieren tabla de referencia? (spike).
- ¿La agregación platform-wide de `ai_usage`/`ai_usage_events` (period-first) tiene cobertura de índices suficiente o se requiere índice aditivo (expand-contract)? (spike en tasks 1.1).
- ¿La org dedicada `platform-ops` es viable en el tenant Stytch (slug reservado, membresía de operadores) o Stytch expone un modelo de miembro de plataforma sin org? (spike en tasks 1.1; grounded: docs Stytch — sesiones B2B siempre org-scoped).
- ¿Política de fallo de escritura de `platform_access_log` (fail-open vs fail-closed en lecturas si el insert falla)? (decisión en tasks 2.8 con observabilidad/alertas).
