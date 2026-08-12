# Proposal: Custom roles editor — edición in-app de la política RBAC de Stytch

## Why

El inventario de páginas marca "Custom roles editor (in-app Stytch policy editing — larger change)". Hoy la definición de roles vive solo en el dashboard de Stytch: los admins no pueden crear roles con permisos a medida (p. ej. "solo bandeja + CRM") sin salir del producto. La restricción crítica: **Stytch RBAC policy es el runtime SSOT** — el editor NO crea un segundo SSOT, sino que escribe en la política de Stytch vía su API de roles con validación y versionado. Este change es deliberadamente **gated y posterior**: requiere aprobación de arquitectura/council y el contrato de rollback de política Stytch; es el companion de la página read-only `equipo-permisos` (v1).

## What Changes

- **Editor in-app de roles** (nueva vista `?view=roles` o dentro de `?view=access`, tab avanzado, gate `org:manage` + permiso nuevo `roles:manage`): crear/editar/archivar roles personalizados con nombre, descripción y conjunto de permisos (resource:action con metadata del catálogo `/rbac/roles`); validación en vivo (sin wildcards sueltos, sin permisos desconocidos, sin duplicar roles del sistema).
- **Escritura a la política de Stytch** vía su API de roles (infrastructure adapter `infrastructure/auth/stytch` implementando un domain interface `RolePolicyRepository`; los domain models NO importan el SDK). Todas las escrituras SHALL pasar por el circuito de Stytch (breaker + cache de política) y SHALL ser idempotentes.
- **Roles del sistema protegidos**: `member`/`approver`/`admin` (o los definidos) NO editables ni eliminables; solo se pueden crear/editar/archivar roles personalizados.
- **Versionado y auditoría**: cada cambio de política SHALL registrarse en el audit ledger (quién, qué, cuándo, diff de permisos) — trazabilidad de compliance.
- **Propagación honesta**: tras guardar, nota "Los cambios aplican en hasta 5 minutos" (TTL de caché) y estado de los roles vigentes hasta entonces.
- **Integración con `equipo-permisos`**: los roles personalizados aparecen en la matriz read-only y en el role select de asignación de miembros (la asignación sigue siendo por member API).
- **Backend**: endpoints `/rbac/roles` extendidos con create/update/archive (escritura a Stytch), validación server-side, y catálogo de permisos; tests de idempotencia, breaker abierto (rechazo sin escritura local), y rollback de política.

## Capabilities

### New Capabilities

- `custom-roles-editor`: edición in-app de roles personalizados escribiendo en la política RBAC de Stytch (runtime SSOT) vía API, con roles del sistema protegidos, validación, auditoría y propagación honesta.

### Modified Capabilities

- `stytch-authorization`: se añade la superficie de escritura controlada a la política (roles personalizados vía API con breaker y auditoría), manteniendo el principio de SSOT único — la UI escribe EN Stytch, nunca en una copia local.
- `equipo-permisos`: la matriz read-only y la asignación de miembros SHALL reflejar los roles personalizados definidos.
- `settings-ui`: la navegación incorpora el editor (`?view=roles` o tab avanzado de access) con gate `org:manage` + `roles:manage`.

## Impact

- **Backend**: `go-b2b-starter/` — domain interface `RolePolicyRepository` (sin SDK en domain), adapter `infrastructure/auth/stytch/role_policy.go` (API de roles de Stytch con breaker), endpoints `/rbac/roles` (create/update/archive) con validación, audit de cambios, tests (idempotencia, breaker abierto → sin escritura local, rollback).
- **Frontend**: `next_b2b_starter/app/dashboard/settings/components/` (editor UI), `settings-content.tsx` (vista), `lib/copy/ui.ts`.
- **Auth**: contrato Stytch real — escritura a la política; rollback dual (Git + política Stytch) documentado; permiso nuevo `roles:manage` en la política.
- **Dependencias**: ninguna nueva.
- **Ops**: `make test` (Go), `pnpm build`/`lint`/`tsc`, vitest, Playwright visual/a11y → `qa/`.
- **Rollback**: revert Git + restauración de la política de roles en Stytch (backup/versionado del policy antes de cada escritura; restauración documentada).
- **Non-Goals**: NO editar roles del sistema; NO wildcards sin expansión validada; NO almacenar la política localmente (copia nunca; solo caché de lectura TTL 5 min); NO bypass del breaker; NO asignación de roles fuera de la member API.

## Assumptions

- Stytch B2B expone API para gestionar roles/policy (verificado en documentación pública de referencia; spike en tasks 1.1 confirma métodos concretos y límites).
- "Larger change": requiere gate de council y este proposal se registra como gated; NO se implementa antes de `equipo-permisos` (que define la ventana read-only y el catálogo de permisos).
- El catálogo de permisos editable es el mismo que expone `/rbac/roles` (metadata displayName/description/category).
- El permiso `roles:manage` se añade a la política Stytch y se asigna solo a `admin`.
