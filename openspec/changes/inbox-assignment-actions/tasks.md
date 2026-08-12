# Tasks: inbox-assignment-actions

> Cap: 2h por tarea. Tags: `[FE-NEXT]`, `[BE-INFRA]`. Depende de: `conversation-row-scoping` (PATCH assignee + `inbox:reassign` + auditoría) e `inbox-scope-views` (AssigneeChip, "nueva" dot).

## 1. Picker de tres jugadas [FE-NEXT]

- [ ] 1.1 Componente `AssignmentPicker`: disparado por el chip de cabecera; lista de miembros de la organización (misma fuente que MemberList de `equipo-permisos`); búsqueda; sin indicadores de presencia falsos (mostrar rol, no online).
- [ ] 1.2 Jugada Reclamar: cola → mío (1 click), persiste vía PATCH assignee; toast "Chat reclamado por ti" + acción "Devolver a la cola".
- [ ] 1.3 Jugada Transferir: picker con búsqueda → PATCH assignee al destino; auditoría server-side (ya en el endpoint base).
- [ ] 1.4 Jugada Liberar: confirm dialog + toast de undo 5s (undo solo si el chat sigue en cola; si otro lo reclamó, "Ya fue reclamada por X").
- [ ] 1.5 Gate `inbox:reassign`: sin permiso el chip es read-only (hide, no ghost); 403 server-side si se intenta directo.
- [ ] 1.6 Conflicto 409: claim/transfer concurrentes → toast "Tomado por X" con avatar + refresh del thread.

## 2. Auto-claim al primer reply [BE-INFRA] + [FE-NEXT]

- [ ] 2.1 Backend: UPDATE condicional `WHERE assignee IS NULL` en el POST de mensajes (misma transacción del envío), idempotente; respuesta incluye dueño actual si el claim falló; guardrails intactos (denials en audit).
- [ ] 2.2 Frontend: toast de auto-claim tras enviar en conversación sin asignar + acción "Devolver a la cola".
- [ ] 2.3 Tests de carrera: dos envíos concurrentes → un ganador, ambos mensajes persistidos, perdedor ve dueño actual.

## 3. Banner de ownership en el poll [FE-NEXT]

- [ ] 3.1 Banner en cabecera cuando el assignee cambió desde la apertura ("Reasignada a Ana" / "Reclamada por Ana" / "Devuelta a la cola"); diff por poll (nunca en cada poll); sin bloquear envío.

## 4. Llegada al receptor [FE-NEXT]

- [ ] 4.1 Toast in-app "Conversación asignada a ti" al recibir una conversación (transferencia/reasignación) + punto "nueva" en Mis chats (de `inbox-scope-views`); sin push/email en v1.

## 5. Menú contextual de fila [FE-NEXT]

- [ ] 5.1 "⋯" en fila → "Asignar a…" con el mismo picker (reassign sin abrir el thread); gate `inbox:reassign` + visibilidad; mismo flujo de auditoría.

## 6. Verificación gate [FE-NEXT] + [BE-INFRA] + [OPS-GOV]

- [ ] 6.1 Vitest: picker (tres jugadas, undo, 409, gates), reply-input (auto-claim), header (banner), llegada (toast), menú contextual.
- [ ] 6.2 Go tests (si hay cambio backend en 2.1): auto-claim atómico, carrera, guardrails intactos, idempotencia; `make test`.
- [ ] 6.3 Playwright visual + a11y (390x844 / 768x1024 / 1440x900): picker, banner, toasts → `qa/`.
- [ ] 6.4 Verificación gate final: `make test` (si aplica), `pnpm build`, `pnpm lint`, `npx tsc --noEmit`, vitest, `openspec validate --changes inbox-assignment-actions`; registrar resultados.
