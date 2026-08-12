# Tasks: conversation-row-scoping

> Cap: 2h por tarea. Tags: `[DB-SQLC]`, `[BE-DOMAIN]`, `[BE-INFRA]`, `[FE-NEXT]`, `[OPS-GOV]`.
> Verificación gate: todos los comandos de verificación listados en la sección 6 deben correr y pasar antes de reportar el change completo.
> Rev 2 — responde al VERDICT.md del council (REJECTED, 6 required design changes; ver `revision.md`).

## 1. Política Stytch y contrato de permisos [OPS-GOV]

- [ ] 1.1 Documentar y aplicar en la política Stytch (dashboard/API): permisos `inbox:view_all`, `inbox:view_unassigned`, `inbox:reassign` + asignación a roles (`admin` → los tres; `manager` → `view_all` + `reassign` + `reply`; `member` → ninguno de scope por defecto, salvo `view_unassigned` decidido por producto). Registrar el rollback (remover permisos) en el change.
- [ ] 1.2 Verificar que la política Stytch se lee vía `RBACPolicyService` cacheada (Redis 5-min TTL) y que los permisos nuevos resuelven por wildcard si aplica; **versionar la cache key** (`auth:stytch:rbac:policy:v2`, patrón del action `export`) para que tomen efecto sin esperar el TTL.
- [ ] 1.3 **Espejar los permisos nuevos en el fallback dev/mock**: `internal/modules/auth/rbac.go` (`NewPermission("inbox","view_all")` etc.) y `roles.go` (asignación a Member/Manager/Admin) para paridad mock-auth; el contrato tipado rol→scope (task 3.2) SHALL usarse como tipos de compilación + fallback, NUNCA como fuente runtime (la política cacheada lo es).
- [ ] 1.4 **Directorio de miembros**: servicio Go que lista miembros activos del org vía Stytch B2B `POST /v1/b2b/organizations/members/search` (query vacía, `organization_ids` = org, paginación `next_cursor`, `statuses: [active]`; SDK `stytch/b2b/organizations/members`), envuelto en el circuit-breaker de dos niveles (umbral 5, timeout 10s, probe half-open 2) + cache Redis 5-min TTL (patrón política RBAC); devuelve solo `stytch_member_id`.

## 2. Esquema y migración [DB-SQLC]

- [ ] 2.1 **Migración por pasos — paso 1**: `ALTER TABLE crm.conversations ADD COLUMN assignee_stytch_member_id TEXT` (catalog-only, sin default volátil) + `COMMENT` (FK lógico a Stytch, patrón tickets) + `CREATE INDEX idx_conversations_assignee ON crm.conversations(organization_id, assignee_stytch_member_id)`. Migración down: drop columna e índice.
- [ ] 2.2 **Audit pre-migración** (precedente `000016_pre_migration_audit.sql`): consulta que cuantifica filas que caerán a cola — contactos sin `assigned_to`/`owner_account_id`, y casos `assigned_to`/`owner_account_id` con `accounts.stytch_member_id` NULL — para la comunicación de staffing día uno.
- [ ] 2.3 **Backfill por lotes** (paso 2, UPDATE por batches de id, no full-table único): `UPDATE crm.conversations SET assignee_stytch_member_id = a.stytch_member_id FROM crm.contacts ct JOIN crm.companies co ON co.id = ct.company_id JOIN organizations.accounts a ON a.id = COALESCE(ct.assigned_to, co.owner_account_id) WHERE ...` — prioridad `contact.assigned_to`, fallback `company.owner_account_id`, si ambos NULL o `stytch_member_id` NULL queda NULL (cola). Aplicar solo a conversaciones aún con assignee NULL (idempotente).
- [ ] 2.4 **Índice de ownership** (paso 3): `CREATE INDEX idx_companies_owner ON crm.companies(owner_account_id)` — ejecutar **`CONCURRENTLY`** (runner no-transaccional o paso post-deploy del job; golang-migrate v4 envuelve cada archivo en transacción y lo impide dentro de la transacción envuelta). Verificar en apply el modo del runner.
- [ ] 2.5 **Migración RLS opt-in** (paso 4, flag de deploy, migración aparte): política en `crm.conversations` con predicado de scope (unión) + `SET LOCAL` como mecanismo de session vars + grants de tabla para el rol `app_session` (bypass de INSERT de webhook y de workers). Documentar enable/disable.
- [ ] 2.6 Regenerar modelos SQLC (`make sqlc`) y verificar `go build ./...` sin errores tras la migración.

## 3. Contrato de scope (domain) [BE-DOMAIN]

- [ ] 3.1 Implementar scope resolver `conversation-scope` (domain, sin imports de SDK Stytch ni transportes): predicado de unión `assignee = member OR empresa(contacto) owner = member OR view_all OR (assignee IS NULL AND view_unassigned)`; función única fuente para query layer y RLS.
- [ ] 3.2 Implementar contrato tipado rol→scope: structs Go con permisos de scope (`view_all`, `view_unassigned`, `reassign`) y de acción (`view`, `reply`, `org:manage`) por rol normalizado (patrón `strings.TrimPrefix(roleID, "stytch_")`); **solo tipos de compilación + fallback dev/mock** (la fuente runtime es la política Stytch cacheada, task 1.2).
- [ ] 3.3 Middleware: **abre la transacción del request y setea session vars con `SET LOCAL`** (`app.current_member_id`, `app.is_view_all`, `app.is_view_unassigned`) resuelto desde la política cacheada + contrato; en free tier setear org-wide (flag off). Nunca `SET` a nivel sesión sobre el pool. **Fail-closed + observabilidad**: log/métrica de anomalía si un path interactivo corre con RLS activa y vars ausentes (error 500 explícito, no lista vacía silenciosa). Log de fallo si la política no está disponible (503).
- [ ] 3.4 **Flag `conversation_row_scoping` sobre el entitlement real**: definir el flag en el mapeo plan→features reconciliado (`billingFeatureProvider.GetEntitlement` → `Entitlement.Features`; metadata `crm_features`, módulo vendible o grant base tipo `defaultGrantedModules` — elegir y documentar el canal en el change). Free/inactiva → false. Validar nombres de plan y canal de grant contra el catálogo Polar (supuesto). Resolver el flag una vez por request en el middleware de entitlement.
- [ ] 3.5 **Workers de background**: verificar que outbound send, message-send, campañas, analytics y pipeline durable leen conversaciones bajo rol `app_session` (bypass RLS) o contexto org explícito — nunca contexto de miembro inventado (sin inanición bajo RLS).

## 4. Query layer scoped [DB-SQLC] + [BE-INFRA]

- [ ] 4.1 Auditar TODAS las queries SELECT/UPDATE/DELETE de `crm.conversations` en `query/crm.sql` (~77/81 lista, 90/102 INSERT, 115/123 UPDATE status, 133 stats) y `query/agent.sql` (~156, ~193): aplicar el predicado de scope en los SELECT de lista/thread/poll/unread/stats; los UPDATE de status quedan acotados a filas visibles.
- [ ] 4.2 Endpoint de re-asignación (`PATCH /crm/conversaciones/:id/assignee`): permiso `inbox:reassign`; destino validado contra el directorio (mismo org, activo — task 1.4); auditoría (actor, origen, destino) en audit ledger; 404 fuera de scope; 403 sin permiso; **503 `member_directory_unavailable` si el directorio no está disponible** (circuit abierto / cache vacía).
- [ ] 4.3 Ingestión webhook: auto-match determinístico **org-scoped** (teléfono/NIT → `crm.companies` del org resuelto del `phone_number_id` → `owner_account_id` vía `accounts.stytch_member_id`) en creación de conversación; sin match → `assignee = NULL` (cola); **nunca match cross-tenant**; no sobreescribir assignee en inbounds posteriores.
- [ ] 4.4 Webhook INSERT fuera de scope (rol `app_session`/política RLS permite INSERT); UPDATE/DELETE del webhook restringidos a metadata de sistema (nunca `assignee_stytch_member_id`).
- [ ] 4.5 AI rail: reemplazar scans directos en `agent.sql` por lecturas vía repositorio con predicado de scope; `go build` + regenerar SQLC.

## 5. Frontend — contrato mínimo [FE-NEXT]

> La UI de scope (píldoras, chips, picker, cola, urgencia) vive en changes separados: `inbox-scope-views` (lectura) e `inbox-assignment-actions` (escritura). Este change entrega SOLO el contrato que esos changes consumen.

- [ ] 5.1 Constantes de permisos en `lib/auth/permissions.ts` (`inbox:view_all`, `inbox:view_unassigned`, `inbox:reassign`); gates de sidebar/bandeja usan el contrato rol→scope.
- [ ] 5.2 Extender el DTO de conversación con `assignee_stytch_member_id` y aceptar el param `scope` (mine|queue|all) en la query de lista; paridad unread: el badge consulta con el mismo scope que la lista (test de paridad, sin contadores fantasma).
- [ ] 5.3 La UI respeta el flag `conversation_row_scoping` (free tier sin capa de scope — render gated por flag, listo para que `inbox-scope-views` construya encima).
- [ ] 5.4 Picker de asignación con degradación: si el directorio de miembros no está disponible, estado no-disponible con retry (hide, sin ghost); el thread y el composer permanecen operativos.

## 6. Tests y verificación gate [BE-INFRA] + [FE-NEXT] + [OPS-GOV]

- [ ] 6.1 Go tests: scope resolver (unión, 404 out-of-scope, cola), contrato rol→scope (composición supervisor sin destructivas), session vars por rol y por plan (free vs pago), **reuso de conexión del pool no filtra scope entre miembros (`SET LOCAL`)**, **worker de background bajo RLS no muere en silencio (rol `app_session`)**, reassign + audit (404/403/**503 directorio no disponible**), webhook auto-match **org-scoped** (match mismo-org, **rechazo cross-tenant**, sin-match, no-sobreescritura), webhook bypass RLS (INSERT permitido, UPDATE metadata solo), paridad unread. Correr `make test`.
- [ ] 6.2 Vitest de inbox: tabs por permiso, unread parity, picker de asignación (incl. estado no-disponible con retry), claim desde cola. Correr `pnpm test`/vitest.
- [ ] 6.3 Playwright visual + a11y (390x844 / 768x1024 / 1440x900) de tabs de scope y picker → `qa/` en el change.
- [ ] 6.4 Verificación gate final: `make sqlc`, `make test` (Go), `go build ./...`, `pnpm build`, `pnpm lint`, `npx tsc --noEmit`, `openspec validate --change conversation-row-scoping`; registrar resultados en tasks.md. Verificar audit pre-migración (task 2.2) corre y cuantifica la cola. Registrar VERDICT del council en tasks.md.
