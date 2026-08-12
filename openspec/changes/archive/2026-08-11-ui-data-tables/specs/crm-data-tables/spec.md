## ADDED Requirements

### Requirement: CRM tables paginate

The Contactos, Empresas, and Negocios list views SHALL paginate their result sets using the existing `limit`/`offset` query parameters of the CRM API, with page controls showing current page, page size, and total count. Changing pages SHALL reset the query offset and scroll the table into view.

#### Scenario: Contacts table shows page controls

- **WHEN** an org has more rows than the page size
- **THEN** the Contactos table SHALL render page controls and SHALL request rows via `limit`/`offset`

#### Scenario: Next page loads the following rows

- **WHEN** user clicks the next-page control
- **THEN** the table SHALL fetch the next `offset` window and render those rows

### Requirement: CRM tables sort by column

Contact and company tables SHALL provide sortable columns with visible sort affordances and `aria-sort` attributes. Clicking a column header SHALL toggle ascending/descending sort on the fetched page. Default sort SHALL be stable and documented (e.g., newest first).

#### Scenario: Sorting by name toggles order

- **WHEN** user clicks the Nombre column header
- **THEN** the table SHALL reorder rows by name, toggling ascending/descending on repeat clicks
- **AND** the header SHALL carry the corresponding `aria-sort` value

#### Scenario: Sort state is visible

- **WHEN** a column is sorted
- **THEN** the header SHALL display a sort direction indicator

### Requirement: Row selection with bulk actions

Contact and company tables SHALL support row selection via per-row checkboxes and a select-all header checkbox. With one or more rows selected, SHALL be hidden behind a bulk-actions bar offering bulk delete and bulk export. Bulk delete SHALL delete each selected row sequentially via the existing per-item `DELETE` endpoints, SHALL show per-row failure counts, and SHALL confirm via the custom `ConfirmDialog`. Bulk export SHALL export the selected rows via the existing export endpoints.

#### Scenario: Selecting rows shows the bulk bar

- **WHEN** user selects two contacts
- **THEN** a bulk-actions bar SHALL appear showing the selection count with "Eliminar" and "Exportar" actions

#### Scenario: Bulk delete confirms then deletes each row

- **WHEN** user confirms bulk delete of selected contacts
- **THEN** the app SHALL call `DELETE /api/crm/contactos/:id` for each selected contact sequentially
- **AND** SHALL report the number deleted and any failures, then refresh the list

#### Scenario: Select-all selects the visible page

- **WHEN** user checks the header select-all checkbox
- **THEN** all rows on the current page SHALL be selected

### Requirement: Large tables virtualize

Contact and company tables SHALL virtualize the tbody via `@tanstack/react-virtual` when the rendered row count exceeds a threshold (e.g., 100 rows), rendering only visible rows while preserving scroll height, keyboard navigation, and row selection.

#### Scenario: Virtualized rendering above threshold

- **WHEN** a table holds more than the threshold number of rows
- **THEN** only the visible window of rows SHALL be mounted in the DOM and scrolling SHALL behave like a normal table

### Requirement: Table rows are keyboard-accessible

Table rows SHALL be focusable (tabIndex) and activate on Enter/Space, with `role`/`aria` semantics preserved. Row clicks SHALL remain supported for pointer users.

#### Scenario: Row opens detail with keyboard

- **WHEN** a row is focused and user presses Enter
- **THEN** the detail view SHALL open for that row

#### Scenario: Checkboxes remain individually focusable

- **WHEN** a table row is focused
- **THEN** the row's checkbox SHALL be reachable and toggleable via keyboard

### Requirement: CRM lists use skeletons and distinct empty states

CRM and ticket lists SHALL render `Skeleton` rows while loading (no "Cargando..." text), SHALL render a distinct "no results for the current search/filter" state when the list is non-empty but filters match nothing, and SHALL keep the current empty-data state for truly empty lists.

#### Scenario: Loading renders skeleton rows

- **WHEN** a CRM or ticket list query is pending
- **THEN** the view SHALL render skeleton rows in place of the table body

#### Scenario: Search with no matches shows no-results state

- **WHEN** the list has data but the active search/filter matches nothing
- **THEN** the view SHALL display a no-results message (e.g., "No hay resultados para la búsqueda") distinct from the empty-data state, with a clear-filter action

#### Scenario: Empty list keeps empty-data state

- **WHEN** the org has no rows for the entity
- **THEN** the view SHALL display the existing empty-data message

### Requirement: Kanban stage moves are optimistic

Moving a deal card to another etapa SHALL update the kanban immediately (optimistic cache update) and SHALL roll back to the previous stage on failure, showing a Spanish error toast. The kanban SHALL support keyboard drag via `KeyboardSensor` (in addition to pointer drag), and SHALL retain the per-card "Mover a..." select as a fallback.

#### Scenario: Drag updates column immediately

- **WHEN** user drags a deal card to another column
- **THEN** the card SHALL appear in the target column immediately without waiting for the refetch

#### Scenario: Failed move rolls back

- **WHEN** `PUT /api/crm/negocios/:id/etapa` fails after the optimistic update
- **THEN** the card SHALL return to its original column and a Spanish error toast SHALL be shown

#### Scenario: Keyboard drag moves a card

- **WHEN** a keyboard user focuses a card's drag handle and moves it with the keyboard
- **THEN** the card SHALL move to the target column via `KeyboardSensor` and persist as with pointer drag

### Requirement: Tables use the shared table primitives

CRM and ticket tables SHALL be built on the shared `components/ui/table.tsx` primitives, and the export flow SHALL use a single shared helper (no per-view copy-paste).

#### Scenario: CRM table renders with shared primitives

- **WHEN** a CRM table renders
- **THEN** it SHALL use the shared `ui/table` primitives for structure

#### Scenario: Export uses shared helper

- **WHEN** any list view exports CSV
- **THEN** it SHALL call the shared export helper used by all views
