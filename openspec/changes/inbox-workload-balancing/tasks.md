# Tasks: inbox-workload-balancing (GATED — no implementar antes de los cambios base)

> Cap: 2h por tarea. Tags: `[DB-SQLC]`, `[BE-INFRA]`, `[FE-NEXT]`, `[OPS-GOV]`.
> **GATE**: requiere `conversation-row-scoping`, `inbox-scope-views` e `inbox-assignment-actions` archivados + council APPROVED (routing.json). El contenido es planning anticipado; NO se ejecuta hasta levantar el gate.

## 1. Política Stytch y configuración [OPS-GOV]

- [ ] 1.1 Documentar y aplicar en la política Stytch: permiso `inbox:manage_limits` asignado a `admin`; rollback dual documentado (Git + política).
- [ ] 1.2 (Al levantar el gate) registrar VERDICT del council en tasks.md.

## 2. Backend: límites y carga [DB-SQLC] + [BE-INFRA]

- [ ] 2.1 Migración: tabla `inbox_assignment_limits(organization_id, stytch_member_id NULL=default, role, max_active)` + migración down; defaults por rol (admin=∞, supervisor=15, agente=8).
- [ ] 2.2 Conteo de carga por miembro: conversaciones `assignee = miembro AND status='active'` en queries de lista (SQLC, mismo predicado de scope).
- [ ] 2.3 Endpoints CRUD de límites (gate `inbox:manage_limits`), 403 sin permiso; override individual reemplaza default de rol.

## 3. Frontend: indicadores y vista de equipo [FE-NEXT]

- [ ] 3.1 Progreso "6/8" en el picker de asignación (junto al destinatario) + ámbar al límite + confirmación explícita para transferir igual.
- [ ] 3.2 Progreso en filas de "Todos" (solo `inbox:view_all`/`org:manage`); vista de workload de equipo read-only (conteo por miembro).
- [ ] 3.3 Auto-claim de cola urgente al límite: procede y registra exceso en audit (la ventana 24h manda).
- [ ] 3.4 Gate por flag `conversation_row_scoping` (free tier sin ninguna superficie de workload).

## 4. Verificación gate [FE-NEXT] + [BE-INFRA] + [OPS-GOV]

- [ ] 4.1 Go tests: defaults vs override, conteo de activas (cerradas excluidas), 403 sin permiso, exceso en audit.
- [ ] 4.2 Vitest: progreso en picker, ámbar al límite, confirmación, vista de equipo gated, free-tier sin workload.
- [ ] 4.3 Playwright visual + a11y → `qa/`.
- [ ] 4.4 Verificación final: `make test`, `pnpm build`, `pnpm lint`, `npx tsc --noEmit`, `openspec validate --changes inbox-workload-balancing`; registrar resultados.
