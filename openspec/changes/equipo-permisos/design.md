# Design: Equipo y permisos — rights management consolidado

## Context

- Brief de diseño: una página, tres capas — definiciones de rol (Stytch RBAC, read-only), asignación miembro→rol (editable admin), módulos (metadata de plan, resumen); matriz con expansión de wildcards y preview de impacto por superficie.
- Restricción arquitectónica: Stytch RBAC policy es runtime SSOT (Redis-cached, 5-min TTL, 503 si inalcanzable con caché vacía). La página es una ventana a Stytch, no una copia. Editar la política in-app = segundo SSOT (violación AGENTS.md).
- **Hallazgo de revisión (bloqueante, resuelto en este diseño)**: `internal/modules/auth/provider.go` cablea `NewRBACService()` → `defaultRBACService` (definiciones estáticas `AllRoles` de `rbac.go`); el servicio respaldado por Stytch (`StytchRBACService` + `RBACPolicyService` en `adapters/stytch/`, con caché Redis 5-min y expansión de wildcards) existe pero NO está cableado en código no-test. La living spec `stytch-authorization` ya exige `StytchRBACService` como única implementación ("RBACService implementation backed by Stytch policy") — el cableado actual es una violación preexistente que este change corrige. Adicionalmente, `StytchRBACService.GetAllRoles()` descarta `PolicyRole.Description` (deja `RoleInfo.Description` vacío).
- Piezas existentes: `MemberList`/`InviteMember` (role select, ConfirmDialog, last-admin guard, self-protection — con descripciones de rol hardcodeadas en inglés), `RbacRepository.getRoles()` (roles + metadata), `useMembersQuery`, `useModule`, `PERMISSION_GROUPS`, `audit-log-view` (gate `audit:view`), gates de vista `?view=` en `settings-content.tsx` (allowlist).

## Goals / Non-Goals

**Goals:**
- Página consolidada `?view=access` con 3 tabs URL-addressables, registrada en el allowlist de gates con `org:manage`.
- Matriz read-only con metadata + tooltips + filtro, servida desde la política Stytch (no estática).
- Preview de impacto por miembro derivado de la misma lógica de gating real.
- Cambios recientes inline gated por `audit:view` + link a audit.
- Corregir la fuente de datos de `/rbac/roles` (wiring Stytch) y propagar descripciones de rol desde la política.

**Non-Goals:**
- NO editar la política Stytch (matriz read-only; editor es change futuro `custom-roles-editor`).
- NO duplicar toggles de módulos (resumen + links).
- NO eliminar vistas existentes (`?view=members`, `?view=modules` siguen).
- NO refactor del servicio RBAC más allá del cableado y propagación de descripciones (sin tocar DTOs ni el contrato de API).
- NO introducir cachés de roles en el cliente con vida útil mayor a la TTL de la política.

## Decisions

1. **Fuente de datos: cablear el servicio Stytch (opción (a) del veredicto), con fetch por circuit breaker.** `/rbac/roles` SHALL servirse de `StytchRBACService`/`RBACPolicyService` (caché Redis 5-min existente, wildcard expansion existente), cableado en DI bajo el mismo patrón de placeholder-detection que el `AuthProvider` (`auth/cmd/init.go`): credenciales reales → servicio Stytch; placeholders (dev) → fallback estático `defaultRBACService`. El fetch de la política SHALL pasar por el breaker de dos niveles (`platform/stytch.Client.Run`: umbral 5, timeout 10s, half-open 2) — verificado: NI la copia de `modules/auth/adapters/stytch` (cliente crudo `*b2bstytchapi.API`) NI la copia de `platform/stytch` (que usa `s.client.API()` sin `Run`) cubren hoy el fetch con breaker; el diseño corrige ambas rutas al consolidar. Un fallo con breaker abierto o de API → caché si existe, si no vacío con log (UI "política no disponible"; middleware 503 existente). Alternativas descartadas: (b) mantener `defaultRBACService` y rescopear la matriz como "definiciones de backend" — contradice la living spec y haría la matriz divergente del gating real; (c) doble servicio en producción sin fallback — sin valor y con riesgo de confusión.
2. **Consolidación del servicio de política (una sola implementación servida).** Se consolida en `modules/auth/adapters/stytch` (única copia con `GetAllRoles` + expansión de wildcards + cache key versionado `auth:stytch:rbac:policy:v2`; es la consumida por `StytchRBACService`, `token_verifier` y `StytchAuthAdapter.PolicyService()`). La copia duplicada `platform/stytch/rbac_policy.go` (cache key `stytch:rbac:policy`, retorno `[]string`) NO tiene consumidores — verificado: solo su provisión DI (`inject.go`/`cmd/provider.go`); el repositorio de organizations (`stytch_role_repository.go`) consume el `*stytch.Client` con breaker y llama él mismo al endpoint de política, no el servicio duplicado. Por tanto la task 1.4 confirma cero consumidores por grep, retira el duplicado + su provisión DI + el cache key viejo, y conserva el `Client` con breaker (lo usa el org repo y ahora el fetch del servicio consolidado vía `Client.Run`).
3. **Propagación de descripciones de rol.** El wiring SHALL mapear `PolicyRole.Description` → `RoleInfo.Description` (hoy descartado), para que la matriz y el selector de rol muestren descripciones de la política (single source), no boilerplate.
4. **Wildcards: garantizados por el data path, con residual documentado.** `expandWildcardActions` expande `*` a las acciones del recurso definido en la política. Residual: si un permiso wildcard referencia un recurso no definido en la política, se muestra el wildcard literal con nota "permiso amplio" en el tooltip (no se inventa expansión).
5. **Metadata honesta en tooltips.** La metadata de permiso del DTO es genérica ("contact view", "Can view contact", categoría "General") — los tooltips explican el ORIGEN del permiso (rol + `resource:action` + displayName + expansión), no prometen descripciones curadas por recurso. Las descripciones de ROL sí vienen de la política (decisión 3).
6. **Fuente única para el selector de rol.** El tab Miembros consume las opciones + descripciones de la misma query `/rbac/roles` que la matriz (fallback: copy español en `lib/copy/ui.ts`). Se eliminan las descripciones hardcodeadas en inglés de `invite-member.tsx` como fuente primaria.
7. **Gating del ledger inline por `audit:view`.** La sección "Cambios recientes" se renderiza y fetchea solo con `org:manage` AND `audit:view` (mismo predicado que `audit-log-view`); sin `audit:view` se omite sin fetch. Alternativa descartada: definir un permiso derivado nuevo — innecesario, el predicado existente cubre el caso.
8. **Contrato de frescura y fallo de la query de roles.** TanStack Query con `staleTime ≤ 5 min` (igual a la TTL de la política), `refetchOnWindowFocus`, y control de refetch manual en la matriz ("Actualizar"). Lista vacía ≠ sin permisos: estado "política no disponible" con retry. El spike (task 1.1) verifica el modo de fallo real del endpoint (error vs 200 vacío) y se refleja en el estado.
9. **Registro en allowlist.** `?view=access` se añade al allowlist de gates de `settings-content.tsx` con `canManageMembers` (mecanismo existente); sin registro la vista cae a overview (nunca datos sin gate).
10. **Estados** — skeleton (tabla+matriz), error+retry, vacío (CTA invitar), 403 sin datos, 401 → login (flujo estándar, `skipAuth` nunca usado).

## Compliance con veredicto del consejo

| # | Veredicto (cambio requerido) | Resolución en este diseño |
|---|---|---|
| 1 | Resolver premisa de source-of-truth (wiring o rescope con residual) | Decisión 1: wiring Stytch (opción (a)); fallback estático solo dev/placeholders; residual de drift eliminado |
| 2 | Fijar expansión de wildcards al data path activo | Decisiones 1+4: expansión garantizada por el servicio Stytch; residual de wildcard literal documentado |
| 3 | Gatear ledger inline por `audit:view` | Decisión 7 + escenario en delta spec |
| 4 | Contratos de frescura y fallo de la query | Decisión 8: `staleTime ≤ 5 min`, refetch manual, estado indisponibilidad |
| 5 | Fuente única de descripciones de rol | Decisiones 3+6: política Stytch → matriz y selector; fallback copy español |
| + | Allowlist de gates (`?view=access`) | Decisión 9 |

**Re-revisión (veredicto 2):**

| # | Cambio requerido | Resolución |
|---|---|---|
| 1 | Conciliar fallback dev con living spec "DTOs retained as API contract" | Delta spec MODIFICA "DTOs retained as API contract": eliminación de datos estáticos acotada al data path de producción; fallback dev permitido solo con gate de placeholder-detection |
| 2 | Cobertura de circuit breaker + consolidación del path de política | Decisiones 1+2: fetch vía `Client.Run` (breaker 5/10s/2), única implementación servida (`auth:stytch:rbac:policy:v2`), retiro del duplicado `platform/stytch` y su cache key |

**Revisión 3 (veredicto 3):**

| # | Cambio requerido | Resolución |
|---|---|---|
| 1 | Corregir premisa de consumidores de la consolidación | Decisión 2 + task 1.4: duplicado sin consumidores (solo provisión DI); retiro de duplicado + cache key; conservar `Client` con breaker (org repo + fetch consolidado vía `Run`) |
| 2 | Renumerar decisiones de design.md | Decisiones renumeradas secuencialmente 1–10; referencias cruzadas y tabla de compliance corregidas |

## Risks / Trade-offs

- [Matriz desactualizada por TTL 5 min] → Mitigación: `staleTime ≤ 5 min` + refetch manual + nota de propagación; la política es runtime SSOT.
- [Política Stytch inalcanzable con caché vacía] → Mitigación: breaker de dos niveles en el fetch (`Client.Run`); `GetAllRoles` retorna vacío con log; la matriz muestra "política no disponible" con retry; auth middleware mantiene 503 (contrato existente).
- [Wildcard sobre recurso no definido en la política] → Residual aceptado: celda muestra wildcard literal con nota "permiso amplio"; sin inventar expansión.
- [Fallback estático en dev diverge de producción] → Residual aceptado: solo credenciales placeholder (dev); gateado por el mismo chequeo que el AuthProvider; documentado en copy "Fuente: política Stytch" solo en producción.
- [Preview se desvía del gating real] → Mitigación: compartir la función de gating (una fuente); si no es factible, el preview marca "basado en la configuración actual".
- [Carga de roles falla (backend)] → Mitigación: error + retry; la matriz no bloquea los tabs de miembros.

## Migration Plan

1. **Backend (`[BE-INFRA]`)** — cablear `StytchRBACService`/`RBACPolicyService` en el provider del módulo auth (patrón placeholder-detection), fetch de política vía `Client.Run` (breaker 5/10s/2), propagar `PolicyRole.Description`, log en fallo; consolidar el servicio de política (retirar duplicado `platform/stytch` + su provisión DI + cache key viejo — sin consumidores). Verificación: `go test ./internal/modules/auth/...` (+ path breaker con fallo simulado); `GET /api/rbac/roles` devuelve roles de la política.
2. **Frontend** — crear `equipo-permisos.tsx` (+ componentes: matriz, preview, módulos resumen, cambios recientes) en `app/dashboard/settings/components/`.
3. **Integración** — `view=access` en allowlist de `settings-content.tsx` (gate `org:manage`, overview section, URL `tab`); selector de rol con fuente única; sección ledger con gate `audit:view`.
4. **Copy** — `lib/copy/ui.ts` (español tipado): "Equipo y permisos", "Matriz de permisos", "Fuente: política Stytch", "Cambios aplican en hasta 5 minutos", "Último admin — no se puede quitar".
5. **Gates** — lint/build/tsc + vitest (member-list, nuevo access-tab) + visual Playwright; `go test` del módulo auth.
6. **Rollback** — git revert; el wiring vuelve a `NewRBACService()` sin estado de DB/Stytch (la página no escribe política).

## Open Questions

- ¿El gating de la navegación es extraíble a un helper compartido sin refactor grande? (si no, el preview usa la misma lógica inline marcada como tal).
- ¿El audit log expone eventos de cambio de rol con actor/sujeto/fecha? (verificar endpoint en tasks; si no, cambios recientes usa el ledger existente).
- Resuelto: expansión de wildcards (garantizada por el data path Stytch, decisión 4); metadata de permiso genérica (decisión 5).
- Resuelto en re-revisión: cobertura de breaker y consolidación del path de política (decisiones 1+2); coherencia con "DTOs retained as API contract" (delta spec MODIFICADO).
- Resuelto en revisión 3: premisa de consumidores corregida (decisión 2/task 1.4); decisiones renumeradas (1–10).
