# Proposal: conversation-row-scoping — visibilidad de conversaciones por menor privilegio (planes pagos)

## Why

Hoy la fila de una conversación se acota solo por `organization_id`: cualquier miembro con el permiso de bandeja (`inbox:view`/`inbox:reply` tras `inbox-member-tier`, u `org:manage` hoy) ve TODAS las conversaciones del tenant. Para un SaaS B2B esto viola el principio de menor privilegio: un SDR debería ver solo los chats de sus cuentas, no la cola completa de la organización. La infraestructura de ownership ya existe (`crm.companies.owner_account_id`, `crm.contacts.assigned_to`, patrón `assignee_stytch_member_id` en tickets) pero las conversaciones nunca se conectaron a ella. Este change introduce scoping por fila derivado del grafo de ownership (Modelo B: gestión por cuenta B2B), con cola de no-asignados como fallback para leads net-nuevos, enforcement en query layer + RLS, y está **restringido a planes pagos** vía feature gate.

## What Changes

- **Columna nueva** `crm.conversations.assignee_stytch_member_id` (TEXT, patrón de `crm.tickets` — FK lógico a Stytch, sin tabla local de miembros). `NULL` = conversación no asignada.
- **Scope resolver** (regla de unión): una conversación es visible para un miembro si:
  - `assignee_stytch_member_id = member` (asignación directa), **O**
  - el contacto pertenece a una `crm.companies` con `owner_account_id` cuyo `stytch_member_id` = member (ownership de cuenta, unión con `contact.assigned_to`), **O**
  - el miembro tiene `inbox:view_all` (scope org-wide explícito, composable con acciones), **O**
  - la conversación está sin asignar y el miembro tiene `inbox:view_unassigned` (cola de leads).
- **Permisos nuevos en la política Stytch** (runtime SSOT): `inbox:view_all` (scope org-wide), `inbox:view_unassigned` (ver cola de no-asignados), `inbox:reassign` (re-asignar). Composición ortogonal: `view_all` + `inbox:reply`/`inbox:reassign` construyen un "supervisor" sin capacidades destructivas (`inbox:delete`/`org:manage` NO se conceden por el scope).
- **RLS extendido** (`lean-data-isolation`): session vars `app.current_member_id`, `app.is_view_all`, `app.is_view_unassigned` (seteadas por el middleware Go **con `SET LOCAL` dentro de la transacción del request** — nunca `SET` a nivel sesión sobre el pool — y resueltas desde la política Stytch cacheada; los structs Go del contrato rol→scope son tipos de compilación + fallback dev/mock) + política RLS en `crm.conversations` con el mismo predicado de scope (defense-in-depth, opt-in; primera implementación real de RLS — los workers de background corren con rol `app_session`/contexto org explícito para no morir en silencio bajo RLS).
- **Query layer**: TODA consulta SELECT sobre `crm.conversations` (lista, thread, poll 5s, unread counts, stats, AI en `agent.sql`) gana el predicado de scope. El mayor riesgo de fuga: unread/paginación — auditados explícitamente.
- **Ingestión webhook**: INSERT de conversaciones/contactos SIEMPRE permitido (bypass de scope con rol `app_session` o política RLS que permita INSERT del servicio de webhook); UPDATE/DELETE por webhook restringidos a metadata de sistema. Routing deterministico **org-scoped**: antes de caer a la cola, intentar match de contacto/empresa por teléfono o NIT **dentro del org resuelto del `phone_number_id` del payload** (`whatsapp.whatsapp_configs.phone_number_id` es UNIQUE); si hay match, auto-asignar a `owner_account_id`; si no, `assignee = NULL` → cola visible para `inbox:view_unassigned`.
- **AI Rail hereda el scope**: `ai-context-intelligence` NO lee `crm.conversations` por scan directo (`agent.sql`); consulta a través del repositorio con predicado de scope, de modo que el contexto LLM queda acotado por lo que el miembro ve.
- **Migración determinística por pasos** (sin grace-period flag): (a) `ADD COLUMN` catalog-only; (b) audit pre-migración que cuantifica la cola (incl. `assigned_to`/`owner_account_id` con `accounts.stytch_member_id` NULL); (c) backfill por lotes: `assignee_stytch_member_id = contacts.assigned_to` (join); si NULL, fallback a `companies.owner_account_id` (join a `accounts.stytch_member_id`); si aún NULL, queda NULL → cola; (d) `CREATE INDEX CONCURRENTLY idx_companies_owner` (runner no-transaccional o post-deploy). Comunicación a admins para staffing de la cola el día uno.
- **Feature gate**: la visibilidad restringida solo se activa en planes pagos, vía el mecanismo de entitlement real (`FeatureProvider.GetEntitlement` → `Features[conversation_row_scoping]`; metadata de suscripción / módulo / grant base, patrón `defaultGrantedModules`). En free tier (sin suscripción o inactiva), el flag es false y el comportamiento actual (org-scope) permanece intacto.
- **UI (inbox-ui)**: contrato mínimo en este change (constantes, gates, DTO con `assignee`, param `scope`, flag-aware). Las superficies de UI — píldoras de scope/cola/todos, chips de assignee, picker de claim/transfer/release, urgencia de cola — se entregan en changes separados: `inbox-scope-views` (lectura) e `inbox-assignment-actions` (escritura); `inbox-workload-balancing` (capacidad) queda gated/futuro.

## Capabilities

### New Capabilities

- (ninguna — el scoping se gobierna por deltas en capacidades existentes)

### Modified Capabilities

- `stytch-authorization`: permisos nuevos `inbox:view_all`, `inbox:view_unassigned`, `inbox:reassign` en la política Stytch; mapeo rol→scope tipado (contrato governance: roles Stytch → contextos locales de autorización con tipos estrictos); enforcement server-side.
- `whatsapp-inbox`: lista/thread acotados por el scope resolver (assignee ∪ owner ∪ view_all ∪ cola); la bandeja deja de ser org-wide para roles sin `view_all`.
- `lean-data-isolation`: extensión del patrón RLS a nivel miembro (`app.current_member_id`, `is_view_all`, `is_view_unassigned`) con política en `crm.conversations`; bypass de ingestión documentado.
- `whatsapp-webhook-ingress`: auto-match determinístico (teléfono/NIT → empresa → `owner_account_id`) en el path de ingestión; caída a cola de no-asignados cuando no hay match; INSERT siempre permitido bajo RLS.
- `ai-context-intelligence`: el contexto de conversación se obtiene a través del repositorio con scope aplicado; sin scans directos de `crm.conversations`.
- `feature-gating`: flag nuevo (p. ej. `conversation_row_scoping`) activo solo en planes pagos; free tier conserva org-scope.
- `inbox-ui`: tabs de visibilidad (Mine/Unassigned/All), picker de asignación/re-asignación gated por `inbox:reassign`, copy de cola y estados vacíos.

## Impact

- **Backend** (`go-b2b-starter/`):
  - Migración SQL por pasos: `ALTER TABLE crm.conversations ADD COLUMN assignee_stytch_member_id TEXT` + audit pre-migración + backfill por lotes + `CREATE INDEX CONCURRENTLY idx_companies_owner` + política RLS opt-in.
  - SQLC: queries en `query/crm.sql` (líneas ~77/81 lista, 90/102 INSERT, 115/123 UPDATE status, 133 stats) y `query/agent.sql` (~156, ~193 AI) — predicado de scope en todos los SELECT; UPDATE/DELETE scoped.
  - Nuevo scope resolver (domain `conversation-scope`), session-var writer en middleware, política RLS en migración (opt-in).
  - Entitlement provider (`billingFeatureProvider.GetEntitlement`): grant del flag `conversation_row_scoping` para planes pagos (mecanismo reconciliado con el drift spec↔código — el spec vivo describe `plans.go` que no existe en el código).
  - Webhook ingest: auto-match + bypass RLS (rol `app_session`).
- **Frontend** (`next_b2b_starter/`): `app/dashboard/inbox/*` (tabs de scope, picker de asignación), `lib/auth/permissions.ts` (constantes nuevas), `lib/copy/ui.ts`, sidebar/gates (la bandeja se ve con `inbox:view`/`inbox:view_all`/`inbox:view_unassigned` u `org:manage`).
- **Auth**: cambio de contrato Stytch real — permisos nuevos en la política (runtime SSOT) + outbound nuevo (Stytch Members API para el directorio de reasignación, con circuit-breaker de dos niveles y cache Redis); rollback dual documentado (Git + política Stytch).
- **Dependencias**: ninguna nueva.
- **Ops**: `make sqlc`, `make test` (Go), `pnpm build`/`lint`/`tsc`, vitest de inbox, Playwright visual/a11y → `qa/`; comunicación de despliegue a admins (staffing de cola, con cola cuantificada por el audit pre-migración).
- **Rollback**: git revert + reversión de la política Stytch (remover permisos `inbox:view_all`/`inbox:view_unassigned`/`inbox:reassign`) + drop de columna/índice/política RLS (migración down) + revertir el grant del flag; sin estado de Stytch local.

## Assumptions

- **Nombres de plan "Starter/Pro/Enterprise"** y el canal de grant del flag `conversation_row_scoping` (metadata de suscripción vs módulo vendible vs grant base) → validar contra el catálogo de Polar y el entitlement provider durante apply; el spec vivo de `feature-gating` describe un `FeatureService`/`plans.go` que no existe en el código (drift spec↔código preexistente) y este change lo reconcilia.
- **Contrato de la Stytch B2B Members API**: validado en docs oficiales de Stytch — `POST /v1/b2b/organizations/members/search` (query vacía + `organization_ids`, paginación `next_cursor`, `statuses: [active]`; SDK Go `stytch/b2b/organizations/members`); verificar la versión exacta del SDK pinneado en apply. El directorio se consume con circuit-breaker de dos niveles + cache Redis y degradación visible (503 `member_directory_unavailable`).
- **`inbox-member-tier` (change en curso)** introduce `inbox:view`/`inbox:reply`; este change compone sobre ese contrato (capacidad × scope) sin depender de su implementación final.
- **golang-migrate v4 envuelve cada archivo de migración en transacción** (verificado en go.mod v4.17.1) → `CREATE INDEX CONCURRENTLY` requiere modo no-transaccional del runner o paso post-deploy.

## Non-Goals

- NO almacenar credenciales/sesiones localmente (todo auth sigue en Stytch B2B; la columna es FK lógico `stytch_member_id`, nunca datos de sesión).
- NO tabla genérica de equipos (`crm.teams`/`crm.team_members`) en este change — el scope es entidad (empresa) + asignación individual; equipos quedan como iteración futura.
- NO round-robin ni routing automático por carga para leads net-nuevos (se rechaza explícitamente: los leads B2B sin calificar no se reparten en rueda).
- NO grace-period con feature flag temporal para la migración (backfill determinístico, sin lógica temporal en RLS/query).
- NO `inbox:delete` ni ampliación de `org:manage` — el scope nunca concede acciones destructivas.
- NO cambio en guardrails de envío (`agent-governance`), streaming/SSE, ni consentimiento.
