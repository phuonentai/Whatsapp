# Tasks: Admin panel — superficie de operador (orgs, uso IA, auditoría cross-org)

Shell de operador con vistas read-only cross-org, gate `platform:operate`. Backend Go + frontend Next. Orden: spike/política → backend → frontend → verificación.

## 1. Spike y política [BE-INFRA]

- [ ] 1.1 Spike: (a) verificar fuentes de estado de suscripción por org (tabla/endpoint); (b) agregación de `ai_usage` por org/periodo: forma de query y COBERTURA DE ÍNDICES para agregación platform-wide (period-first) sobre `ai_usage`/`ai_usage_events` + límites de paginación; (c) fuente de tasas de modelo (configuración vs tabla); (d) modelo de operador Stytch: org dedicada `platform-ops` viable (slug reservado) vs alternativa de miembro sin org, y claim `roles`/custom claim `{{ member.rbac.platform.actions }}` en el JWT. Verify: notas en tasks; fuentes confirmadas; decisión de índice (expand-contract si aplica) documentada
- [ ] 1.2 Añadir rol/permiso `platform:operate` a la política Stytch (solo operadores, org de plataforma dedicada); documentar rollback. Verify: política aplicada; rollback documentado
- [ ] 1.3 Migración aditiva `platform_access_log` (append-only: actor_stytch_member_id, actor_stytch_organization_id, target_organization_id nullable, action, created_at; retención 90 días configurable; SIN ALTERs sobre tablas existentes) + job de limpieza de retención (borrado >90 días, índice `(created_at)`). Verify: migración up/down limpia; `make test`

## 2. Backend Go [BE-DOMAIN]

- [ ] 2.1 Middleware de contexto de plataforma: grupo `/api/v1/platform/*`, autenticación JWT existente (JWKS edge / `X-Forwarded-Auth`), resolución de `platform:operate` vía política cacheada + roles del JWT (sin `authorization_check` para lecturas cross-org), principal de plataforma sin tenant, validación de `org_id` contra `organizations` (400/404). Verify: `make test`; tests por rol
- [ ] 2.2 Endpoint lista de orgs con estado (nombre, membresía, suscripción, conexiones) + búsqueda/paginación, gate `platform:operate`; SIN datos de CRM/contactos/conversaciones. Verify: `make test`; tests 200/403
- [ ] 2.3 Endpoint uso IA por org/periodo (agregado de `ai_usage`, % vs `ai_credits_max`) + filtros + paginación obligatoria server-side. Verify: `make test`; agregación correcta; límites aplicados
- [ ] 2.4 Endpoint auditoría cross-org (eventos del ledger `ai_usage_events` + eventos operativos, filtrables por org/tipo/fecha), read-only y acotado a datos operativos. Verify: `make test`; sin mutación; sin datos CRM en respuesta
- [ ] 2.5 Endpoint tasas de modelo (referencia read-only). Verify: `make test`
- [ ] 2.6 Gate `platform:operate` en todos los endpoints del shell (403 sin permiso; 503 si política no disponible y caché vacía). Verify: `make test`; tests por rol + fallback 503
- [ ] 2.7 Regresión de no-fuga cross-org: un operador NO puede leer datos de otra org vía endpoints member-scoped (scoping intacto); un miembro sin permiso recibe 403 en rutas de plataforma. Verify: `make test`
- [ ] 2.8 Registro de acceso de plataforma: toda lectura cross-org (listado/filtro/detalle) inserta en `platform_access_log` (actor, org objetivo, acción, timestamp). Verify: `make test`; filas presentes; sin datos de negocio en el log

## 3. Frontend [FE-NEXT]

- [ ] 3.1 Shell `/admin` con layout/sidebar de plataforma (Organizaciones, Uso IA, Auditoría — SIN Siigo) y gate `platform:operate` (403 sin permiso; 503 → "política no disponible"; sin enlaces para miembros). Verify: `pnpm build`; 403 y navegación
- [ ] 3.2 Vista Organizaciones: tabla con búsqueda/paginación, estado de suscripción y conexiones WhatsApp/Instagram; detalle de org con estado de integraciones y uso IA (sin Siigo, sin actividad CRM/contactos). Verify: `pnpm build`
- [ ] 3.3 Vista Uso IA: tabla por org (tokens, créditos, % vs límite), filtros periodo/org, tablas de tasa de modelo. Verify: `pnpm build`; sin datos → 0/"—"
- [ ] 3.4 Vista Auditoría cross-org: eventos del ledger con filtros (org/tipo/fecha), read-only y sin datos de clientes. Verify: `pnpm build`
- [ ] 3.5 Siigo fuera de alcance: sin sección ni datos Siigo en la plataforma (sin enlaces, sin estado, sin credenciales); `siigo-admin-view` permanece en settings del tenant SIN CAMBIOS (coordinación con `settings-redesign`). Verify: `pnpm build`; `npx vitest run app/dashboard/settings/components/siigo-admin-view.test.tsx` (regresión: sigue pasando sin cambios)
- [ ] 3.6 Copy nueva en `lib/copy/ui.ts` (español tipado: "Organizaciones", "Uso IA", "Tasas de modelo", "Auditoría"). Verify: `pnpm lint`

## 4. Verificación [OPS-GOV]

- [ ] 4.1 Gate backend: `make test` (incluye 403 por rol, fallback 503, agregaciones y regresión de no-fuga), `make lint`. Verify: comandos PASS
- [ ] 4.2 Gate estático frontend: `pnpm lint`, `pnpm build`, `npx tsc --noEmit`. Verify: los tres comandos PASS
- [ ] 4.3 Gate unitario frontend: `npx vitest run app/admin/` (si aplica) + settings siigo-admin (sin cambios, regresión) → PASS. Verify: comando PASS
- [ ] 4.4 Gate visual/a11y: capturas Playwright 390x844/768x1024/1440x900 del shell y vistas → `openspec/changes/admin-panel-surface/qa/`; tablas con `th scope="col"`, estados con texto. Verify: artefactos en `qa/`
- [ ] 4.5 Cierre: `openspec validate admin-panel-surface --type change` PASS; registrar decisión de archivado (ejecutar /opsx-archive o anotar `**Archive deferred:** <razón>`). Verify: comando validate PASS
