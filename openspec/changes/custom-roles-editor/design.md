# Design: Custom roles editor — edición in-app de la política RBAC de Stytch

## Context

- `equipo-permisos` define la ventana read-only a la política; este change la complementa con escritura controlada de roles personalizados.
- Restricción: Stytch RBAC policy = runtime SSOT (Redis-cached 5-min TTL; 503 si inalcanzable con caché vacía). El editor escribe EN Stytch vía API — no crea un segundo SSOT.
- Clean Architecture: domain models NO importan SDKs; el adapter `infrastructure/auth/stytch` implementa interfaces de dominio; todas las llamadas a Stytch pasan por el breaker de dos niveles (threshold 5, timeout 10s, half-open probe 2).
- Este change es gated: requiere council y se implementa DESPUÉS de `equipo-permisos` (que fija el catálogo y la ventana read-only).

## Goals / Non-Goals

**Goals:**
- Editor de roles personalizados con validación y catálogo de permisos.
- Escritura a Stytch idempotente, breaker-guarded, auditada.
- Roles del sistema protegidos; propagación honesta (5 min).

**Non-Goals:**
- NO editar roles del sistema.
- NO copia local editable de la política (solo caché de lectura).
- NO bypass del breaker ni escritura fuera de Stytch.

## Decisions

1. **Domain interface `RolePolicyRepository`** (en `domain/`) con `CreateRole/UpdateRole/ArchiveRole/ListRoles`; el adapter `infrastructure/auth/stytch/role_policy.go` implementa la API de roles de Stytch con breaker. Alternativa (llamar al SDK desde handlers) se descarta: violaría la regla de que domain no importa SDKs y acoplaría transporte.
2. **Escritura con backup/versionado** — antes de cada escritura se captura el estado de la política afectada (roles + permisos) para restauración; el rollback documenta la restauración vía la API de Stytch + revert de Git.
3. **Validación en dos capas** — UI (catálogo conocido, sin duplicados, nombres válidos) + servidor (permisos del catálogo, roles del sistema protegidos, idempotencia). El servidor es la autoridad.
4. **Roles del sistema por allowlist** — `member`/`approver`/`admin` no editables; los roles personalizados se marcan con flag propio en la política (o convención de nombre) para que la UI los distinga.
5. **Propagación honesta** — nota "Los cambios aplican en hasta 5 minutos" (TTL) + refetch de matriz/asignación; nunca afirmar instantaneidad.
6. **Auditoría** — diff de permisos por operación al audit ledger existente (append-only).
7. **UI** — vista `?view=roles` (o tab avanzado en access) con editor de permisos por categoría (reutilizando metadata de `/rbac/roles`), lista de roles personalizados con estado (activo/archivado), confirmaciones para archivar.

## Risks / Trade-offs

- [Escritura a la política con errores parciales] → Mitigación: idempotencia + backup previo + auditoría; reintentos seguros.
- [Breaker abierto en escritura] → Mitigación: rechazo sin efecto + mensaje claro; la UI no muestra éxito falso.
- [Cache 5 min hace parecer que no aplicó] → Mitigación: nota de propagación + refetch; consistencia eventual aceptada y comunicada.
- [Rol personalizado roto (permiso retirado del catálogo)] → Mitigación: validación al editar; roles archivados no se eliminan en cascada.
- [Cambio grande: scope creep] → Mitigación: gated; solo roles personalizados; asignación y matriz ya existen.

## Migration Plan

1. **Política Stytch**: añadir `roles:manage` (admin); documentar backup/restauración.
2. **Backend**: domain interface + adapter + endpoints create/update/archive + validación + auditoría + tests (idempotencia, breaker, 403 roles del sistema).
3. **Frontend**: editor UI + integración en `?view=access` + copy.
4. **Gates**: `make test`, lint/build/tsc, vitest, Playwright visual/a11y.
5. **Rollback**: revert Git + restauración de la política desde backup (procedimiento documentado en tasks).

## Open Questions

- ¿La API de roles de Stytch soporta crear/actualizar roles con permisos arbitrarios o requiere formato específico? (spike 1.1 antes de implementar).
- ¿Convención para distinguir roles personalizados vs del sistema en la política? (flag vs prefijo; decisión en spike).
- ¿El catálogo de permisos editable coincide con los roles existentes o se expande? (el editor expone el catálogo de `/rbac/roles`).
