# Tasks: inbox-scope-views

> Cap: 2h por tarea. Tags: `[FE-NEXT]`, `[BE-INFRA]` (solo verificación de contrato). Depende del contrato de `conversation-row-scoping` (DTO con `assignee`, param `scope`, flag `conversation_row_scoping`).

## 1. Base de datos de UI (contrato)

- [ ] 1.1 Verificar contrato del change base: el DTO de conversación expone `assignee_stytch_member_id` y la lista acepta el param `scope`; si falta, pausar y coordinar (dependencia explícita).
- [ ] 1.2 `lib/models/conversation.model.ts`: extender el modelo con `assignee` (id + nombre derivado) y `newlyAssignedAt` (para "nueva"); sin romper los consumidores existentes del modelo.

## 2. Gramática de ownership [FE-NEXT]

- [ ] 2.1 Componente `AssigneeChip`: iniciales + color estable derivado del id (hash), slot vacío con anillo ámbar para sin-asignar, anillo de realce para "a ti"; render único usado por lista, cabecera y (futuro) picker.
- [ ] 2.2 Integrar chip en `conversation-list.tsx` (fila) y `conversation-header.tsx` (cabecera) tras el flag; tooltip con nombre completo; sin degradar densidad (chip 16px).
- [ ] 2.3 Punto "nueva" (ownership llegó) diferenciado del badge de no-leído existente; render distinto y coexistentes.

## 3. Selector de scope aditivo [FE-NEXT]

- [ ] 3.1 Píldoras de scope con contadores ("Mis chats (n)" · "Cola (n)" · "Todos (n)") encima de los tabs existentes; visibilidad por permiso (`view_unassigned` → Cola; `view_all`/`org:manage` → Todos); estilo de píldora distinto de tabs (`aria-pressed`, no `role=tab`).
- [ ] 3.2 Estado por píldora: persistir selección en la URL (`?scope=mine|queue|all`) sin tocar los params existentes de estado/canal; las queries usan el mismo contrato del backend.
- [ ] 3.3 Móvil 390px: píldoras de scope permanecen; tabs de estado/canal existentes colapsan a botón "Filtrar" con sheet que replica exactamente los filtros (sin pérdida de funcionalidad).
- [ ] 3.4 Contadores por píldora: fetch de conteos con el mismo predicado de scope (paridad lista-vs-conteo; sin contadores fantasma).

## 4. Estados vacíos y urgencia [FE-NEXT]

- [ ] 4.1 Tres estados vacíos (sin-scope-con-cola → CTA "Reclama de la cola (n)"; cola vacía → refuerzo; sin permiso → control ausente); loading/error existentes intactos.
- [ ] 4.2 Countdown de ventana 24h en filas de cola ("responder en 16h", ámbar→rojo) + sort de cola por (urgencia, antigüedad); badge de píldora discreto con pulso sutil + live-region cuando hay urgencia (sin sonido).
- [ ] 4.3 Métricas por audiencia en `inbox-metrics.tsx`: `view_all`/`org:manage` → strip org-wide existente; resto → mini-stats personales.

## 5. Flag y primer-run [FE-NEXT]

- [ ] 5.1 Gatear TODA la capa de scope con `conversation_row_scoping` (free tier pixel-identical al estado previo); tests de ausencia de los elementos en free tier.
- [ ] 5.2 Primer-run de upgrade: si hay cola > 0 al activar el flag, abrir en píldora Cola + tooltip de una vez explicando las píldoras.

## 6. Verificación gate [FE-NEXT] + [OPS-GOV]

- [ ] 6.1 Vitest: conversation-list (chips, nueva-vs-unread, contadores), header (chip), metrics (split por permiso), urgencia (countdown, sort), píldoras (visibilidad por permiso, persistencia URL), free-tier (ausencia).
- [ ] 6.2 Playwright visual + a11y (390x844 / 768x1024 / 1440x900): píldoras, chips, cola con urgencia, free-tier sin capa → `qa/`.
- [ ] 6.3 Verificación gate final: `pnpm build`, `pnpm lint`, `npx tsc --noEmit`, vitest, `openspec validate --changes inbox-scope-views`; registrar resultados.
