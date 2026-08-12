# Proposal: Equipo y permisos — rights management consolidado (una página, tres capas)

## Why

Hoy los derechos están dispersos: asignación de roles vive en "Team access" (`?view=members`), toggles de módulos en "Modules" (`?view=modules`), y nadie puede ver qué significa realmente un rol. El brief de diseño define una página consolidada "Equipo y permisos" con tres capas — definiciones de rol (política Stytch, read-only), asignación miembro→rol (editable, admin), y módulos (metadata de plan, resumen) — más una matriz de permisos con expansión de wildcards y preview de impacto por superficie. La restricción arquitectónica: la política RBAC de Stytch es el runtime SSOT; la página es una **ventana a Stytch, no una copia** — editar la política in-app crearía un segundo SSOT (violación de AGENTS.md). El editor de roles personalizados queda como change futuro (`custom-roles-editor`).

## What Changes

- **Fuente de datos corregida (revisión de consejo, bloqueante)**: `GET /api/rbac/roles` SHALL servirse del servicio respaldado por la política Stytch (`StytchRBACService` + `RBACPolicyService`, caché Redis 5-min TTL existente), cableado en DI cuando hay credenciales reales; el servicio estático (`defaultRBACService`) queda SOLO como fallback de desarrollo/placeholders (mismo patrón que el AuthProvider en `auth/cmd/init.go`). Las definiciones estáticas en código no son la fuente servida en producción.
- **Nueva vista consolidada** `?view=access` en settings (`/dashboard/settings?view=access`), con tres tabs URL-addressables (`&tab=members|matrix|modules`):
  1. **Miembros** — reutiliza `MemberList`/`InviteMember` existentes (ConfirmDialog, last-admin guard, remover); el selector de rol y sus descripciones usan la MISMA fuente que la matriz (`/rbac/roles`; fallback copy en español) — se eliminan las descripciones de rol hardcodeadas en inglés; nota "los cambios aplican en hasta 5 minutos" (TTL de caché de política); botón "Invitar".
  2. **Matriz de permisos** — read-only, fuente `/rbac/roles` (roles × recursos `resource:action` + metadata); celdas ✓/parcial/— con Tooltip que explica el origen del permiso (rol + `resource:action`, incluida expansión de wildcards `contact:*` → acciones declaradas; wildcard literal con recurso ausente de la política se muestra con nota de permiso amplio); filtro por recurso; columna admin con ancla visual. Contrato de frescura: `staleTime ≤ 5 min` (misma TTL que la caché de política) + refetch manual; lista vacía → estado "política no disponible" con retry (nunca "sin permisos").
  3. **Módulos** — resumen de módulos activos con badge de fuente de plan + links a `?view=modules` (toggles) y `?view=subscription` (upgrade); sin toggles duplicados.
- **Preview de impacto por miembro** ("¿qué ve este miembro?"): selector de miembro → mapa de permisos efectivos a superficies (bandeja, IA, knowledge, settings) generado desde la misma fuente que la navegación (misma lógica de gating), para que no pueda desviarse de la aplicación real.
- **Cambios recientes inline** (quién cambió el rol de quién, cuándo) + link al audit log; la sección SHALL gatearse por `audit:view` además de `org:manage` (mismo predicado que `audit-log-view`) para no filtrar datos de auditoría a admins sin ese permiso.
- **Estados**: skeleton (tabla + matriz), error + retry, vacío (sin miembros → CTA invitar), 403 sin datos, 401 → login (flujo estándar, `skipAuth` nunca usado). `?view=access` se registra en el allowlist de gates existente de `settings-content.tsx`.
- **Sin cambios de contrato de asignación**: la asignación de roles sigue vía member API (Stytch `Members.Update`, acción `stytch.member:update.settings.roles`); la matriz es read-only; los toggles de módulos siguen en `?view=modules`.

## Capabilities

### New Capabilities

- `equipo-permisos`: página consolidada de gestión de derechos — tres capas (definiciones read-only, asignación, módulos), matriz con expansión de wildcards, preview de impacto por miembro, audit inline gated; la página es una ventana de solo lectura a la política Stytch (nunca edita la política).

### Modified Capabilities

- `settings-ui`: la navegación de settings incorpora la vista `?view=access` (nuevo tab URL-addressable registrado en el allowlist de gates con `org:manage`); `?view=members`/`?view=modules` se mantienen como vistas existentes y/o quedan cubiertos por la consolidación sin duplicar estado.
- `stytch-authorization`: se fija la fuente servida de la política — `GET /api/rbac/roles` SHALL estar respaldado por el servicio de política Stytch (runtime SSOT) en producción, con definiciones estáticas solo como fallback de desarrollo; las descripciones de rol SHALL propagarse desde la política Stytch; ninguna UI SHALL editar la política directamente (los cambios de política van por Stytch dashboard o un change futuro `custom-roles-editor`).

## Impact

- **Frontend**: `next_b2b_starter/app/dashboard/settings/components/` (nueva `equipo-permisos.tsx` + componentes de matriz/preview/módulos/cambios-recientes), `settings-content.tsx` (nuevo `view=access` en allowlist + gate `org:manage`; gate `audit:view` para la sección inline), `invite-member.tsx`/`member-list.tsx` (selector de rol desde `/rbac/roles`), `lib/copy/ui.ts` (copy "Equipo y permisos", "Matriz de permisos", "Fuente: política Stytch", "Cambios aplican en hasta 5 minutos", "Último admin — no se puede quitar"), `lib/hooks/queries/` (query de roles con `staleTime ≤ 5 min`).
- **Backend (mínimo, `[BE-INFRA]`)**: cablear `StytchRBACService`/`RBACPolicyService` como `RBACService` en DI (`internal/modules/auth/`), fetch de política vía circuit breaker de dos niveles (`platform/stytch.Client.Run`: umbral 5, timeout 10s, half-open 2), propagar `PolicyRole.Description` → `RoleInfo.Description` (hoy se descarta), log en fallo; consolidar el servicio de política (única implementación servida con cache key `auth:stytch:rbac:policy:v2`; retirar el duplicado `platform/stytch` y su cache key — sin consumidores, solo provisión DI; el repo de organizations usa el `Client` con breaker, no el servicio duplicado). Sin migraciones, sin SQLC, sin cambios de esquema.
- **Auth**: cero cambios de contrato Stytch (lectura: `RBAC.Policy` existente; asignación: member API existente); gates `org:manage` (vista) y `audit:view` (ledger inline).
- **Dependencias**: ninguna nueva (shadcn Tabs/Table/Tooltip/Select/DropdownMenu/ConfirmDialog ya presentes).
- **Ops**: `pnpm build`, `pnpm lint`, `npx tsc --noEmit`; `go test ./internal/modules/auth/...`; vitest de member-list/roles existente; visual/a11y Playwright → `qa/`.
- **Rollback**: git revert; sin estado de DB ni de Stytch (la página no escribe política; el cableado es reversible a `NewRBACService()`).
- **Non-Goals**: NO editor de roles personalizados (change futuro `custom-roles-editor`, gated); NO toggles de módulos duplicados; NO escritura a la política Stytch desde la UI; NO almacenar credenciales localmente (todo auth sigue en Stytch B2B); NO cambios a la asignación fuera de la member API; NO refactor del servicio RBAC más allá del cableado y propagación de descripciones.

## Assumptions

- El contrato de `/rbac/roles` está verificado en código: `RoleDTO` con `id/name/description` y `permissions[].resource/action/display_name/description/category` (`handler.go` → `NewRoleDTO`/`NewPermissionDTO`). La metadata de permiso es genérica ("contact view", "Can view contact", categoría "General") — los tooltips explican el ORIGEN del permiso (rol + `resource:action`), no prometen descripciones curadas; las descripciones de ROL sí provienen de la política Stytch (`PolicyRole.Description`).
- La expansión de wildcards (`contact:*` → acciones declaradas) está implementada en `expandWildcardActions` (`adapters/stytch/rbac_policy.go`) y queda garantizada por el cableado del servicio Stytch (task 1.1); si un recurso no existe en la política, el wildcard se muestra literal con nota de permiso amplio (residual documentado).
- El fetch de la política requiere breaker: verificado que ninguna de las dos copias actuales de `RBACPolicyService` (modules/auth con cliente crudo; platform con `s.client.API()` sin `Run`) pasa hoy por el breaker — el wiring (task 1.2) lo corrige vía `platform/stytch.Client.Run`. Verificado además: el `RBACPolicyService` duplicado de `platform/stytch` no tiene consumidores (solo su provisión DI en `inject.go`/`cmd/provider.go`); `stytch_role_repository.go` (organizations) consume el `*stytch.Client` con breaker y no el servicio duplicado. La task 1.4 confirma cero consumidores por grep antes de retirar el duplicado.
- El "preview de impacto" se genera desde la misma lógica de gating de la navegación (sidebar + gates de vista) para no desviarse de la aplicación real; si no es extraíble sin refactor, se documenta el mapeo explícito marcado como tal.
- `PERMISSION_GROUPS` en `lib/auth/permissions.ts` es el mapa de agrupación actual; el preview lo usa como base y se extiende solo con metadata de `/rbac/roles`.
