# Design: inbox-scope-views — superficie de lectura del scoping (UI aditiva)

## Context

- `conversation-row-scoping` (change base) entrega: columna `assignee_stytch_member_id`, scope resolver (unión), permisos `inbox:view_all`/`inbox:view_unassigned`/`inbox:reassign`, lista scoped con parámetro `scope`, flag `conversation_row_scoping` (solo pagos).
- Bandeja actual (verificada): master-detail con dos filas de tabs — estado (All/Active/Closed/Archived) y canal (All/WhatsApp/Instagram) — lista con preview de 60 chars + timestamp + unread, thread con AI rail, poll 5s, `inbox-metrics.tsx`.
- Restricción dura del brief: **la UI actual permanece intacta**. El scope es una capa aditiva.
- Expert research (Front/Intercom/Zendesk 2025-26): visibilidad por rol (Waiting/Mine vs All), estados de trabajo en vez de read/unread, priorización (SLA) antes que antigüedad, feedback inmediato.

## Goals / Non-Goals

**Goals:**
- Selector de scope aditivo (píldoras con contadores) sin tocar tabs existentes.
- Identidad de ownership (chips, slot ámbar) consistente en lista/cabecera.
- "Nueva" (llegó ownership) ≠ no-leído (llegaron mensajes).
- Tres estados vacíos verdaderos; nunca "no hay datos" cuando es "no es tu data".
- Urgencia de cola (countdown 24h + sort SLA).
- Métricas por audiencia (org vs personal).
- Free tier 100% intacto; primer-run de upgrade aterriza en Cola si hay no-asignadas.

**Non-Goals:** escrituras (picker/claim/release) → `inbox-assignment-actions`; workload → `inbox-workload-balancing`; typing en vivo; reestructurar tabs.

## Decisions

1. **Selector de scope como píldoras aditivas encima de los tabs** — una fila compacta `[Mis chats (4)] [Cola (2)] [Todos (9)]` visualmente distinta de los tabs (píldoras redondeadas con contador, no tabs subrayados), colocada entre el encabezado de página y la fila de tabs de estado existente. Las filas de tabs de estado/canal NO se modifican.
   - **Por qué**: "UI intacta" como requisito; las píldoras con contador hacen doble trabajo (navegación + señal de atención discreta). Expert: visibilidad por rol — cada píldora existe solo si el permiso la habilita.
   - **Alternativas**: (a) scope como 4ª fila de tabs — rechazada (4 dimensiones de tabs = sobrecarga); (b) left rail — rechazada (master-detail sin jerarquía que expresar); (c) dropdown de filtro — pierde el contador visible y la urgencia discreta.
   - **Móvil (390px)**: las píldoras de scope se mantienen (3 elementos caben); los tabs de estado/canal existentes colapsan a un botón "Filtrar" que abre un sheet — este colapso es el ÚNICO cambio permitido a la UI existente, y solo en móvil.
2. **Gramática de ownership en un solo componente** — `AssigneeChip` (iniciales, color estable derivado del id) usado en lista, cabecera, picker y banners; slot vacío con anillo ámbar = sin asignar; anillo de realce = "a ti".
   - **Por qué**: la cobertura de un supervisor se lee "de un vistazo" solo si cada superficie usa la misma gramática.
3. **"Nueva" ≠ no-leído** — punto azul "nueva" (ownership llegó, sin mensajes nuevos) como campo derivado `newly_assigned_at > last_seen_assignee_at` en el DTO; el badge de no-leído existente no cambia. Expert: estados de trabajo, no read/unread.
   - **Alternativa**: reutilizar unread para ambos — rechazada (un chat reasignado sin mensajes nuevos no debe sonar como no-leído).
4. **Estados vacíos por scope** — tres variantes (sin-scope-con-cola → CTA a la cola; cola vacía → refuerzo; sin permiso → el control no existe). El estado de error/loading existente (skeleton 5 items + retry) no cambia.
5. **Urgencia de cola** — countdown de la ventana 24h por conversación sin asignar (ácido a las 8h→16h restantes, rojo bajo 4h) + sort de la cola por `(urgencia, antigüedad)`. Expert: priorizar antes que asignar. El badge de la píldora Cola es discreto hasta que existe urgencia, entonces pulso sutil (1 animación, sin sonido, live-region para a11y).
6. **Métricas por audiencia** — `inbox-metrics.tsx` se bifurca: `view_all`/`org:manage` → strip org-wide actual; resto → mini-stats personales. Mismo backend, distinta presentación.
7. **Flag + primer-run** — free tier: `conversation_row_scoping=false` → sin píldoras, sin chips, sin urgencia (bandeja 100% actual). Al activarse el plan: si `cola > 0` → la píldora Cola queda preseleccionada y un tooltip de una vez explica las tres píldoras ("tus chats, la cola y todas").

## Risks / Trade-offs

- [Píldoras mal leídas como tabs (confusión de dimensión)] → Mitigación: estilo de píldora distinto (radius, contador, sin subrayado de tab activo), a11y con `aria-pressed` vs `role=tab` diferenciados.
- [Fuga de "nueva" o urgencia a free tier] → Mitigación: el flag gatea TODA la capa de scope (un solo punto de activación); tests de free-tier sin ninguno de los elementos.
- [Chips de assignee degradan la densidad de lista] → Mitigación: chip compacto (16px) en la fila, slot ámbar sin texto; el nombre completo vive en el tooltip/cabecera.
- [Sort por urgencia desordena la expectativa de antigüedad] → Mitigación: sort por (urgencia, antigüedad) documentado en el tooltip de la píldora Cola ("ordenado por tiempo de respuesta restante").
- [Móvil: colapsar tabs a "Filtrar" rompe flujos existentes] → Mitigación: el sheet replica exactamente los mismos filtros; cobertura e2e de la bandeja en 390px.

## Migration Plan

1. Depende de `conversation-row-scoping` (contrato: DTO con `assignee`, `scope` param, flag). Coordinar el orden de despliegue: backend primero.
2. Frontend: `AssigneeChip` + píldoras + estados vacíos + urgencia + métricas split, todo gated por flag.
3. Rollback: git revert; con flag off la bandeja es pixel-identical al estado actual.

## Open Questions

- ¿El color del chip se deriva del `stytch_member_id` (hash estable) o de una paleta por rol? → decantarse por hash estable por id (el rol cambia, el color no debería).
- ¿El countdown de urgencia cuenta desde el inbound o desde la última respuesta? → propuesta: desde el último inbound sin respuesta (ventana comercial real).
