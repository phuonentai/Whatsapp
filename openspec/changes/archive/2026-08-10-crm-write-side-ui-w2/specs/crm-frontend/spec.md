## ADDED Requirements

### Requirement: Detail-view routing with id parameter

The system SHALL support detail views via `?view=<entity>&id=<id>` where entity is `contactos`, `empresas`, or `negocios`. Clicking a contact row, company row, or deal card SHALL navigate to the corresponding detail view, and back navigation SHALL return to the list/kanban.

#### Scenario: Contact row navigates to detail view

- **WHEN** a user clicks a contact row in the Contactos view
- **THEN** the URL SHALL update to `?view=contactos&id=<contactId>`
- **AND** the contact detail view SHALL be displayed with the selected contact's profile

#### Scenario: Deal card navigates to detail view

- **WHEN** a user clicks a deal card in the Negocios view
- **THEN** the URL SHALL update to `?view=negocios&id=<dealId>`
- **AND** the deal detail view SHALL be displayed

#### Scenario: Back navigation returns to the list

- **WHEN** a user navigates to a detail view and clicks back
- **THEN** the URL SHALL return to the list view (`?view=<entity>`)
- **AND** the list/kanban SHALL be displayed again

### Requirement: Contact detail view in Spanish

The system SHALL display a contact detail view showing all profile fields (including Tipo Documento and Número Documento), the contact's tags, its associated negocios, and an activity timeline. Action buttons SHALL read "Editar", "Agregar nota", "Crear negocio".

#### Scenario: Contact detail shows documento fields and negocios

- **WHEN** a user opens a contact's detail view
- **THEN** the profile section SHALL display Tipo Documento and Número Documento
- **AND** the detail view SHALL list negocios associated with the contact
- **AND** the activity timeline SHALL display the contact's activities

#### Scenario: Contact detail action buttons in Spanish

- **WHEN** a user opens a contact's detail view
- **THEN** buttons SHALL read "Editar", "Agregar nota", and "Crear negocio"

### Requirement: Company detail view in Spanish

The system SHALL display a company detail view showing the company's profile fields, contact and negocio counts, associated negocios, and an activity timeline, all in Spanish.

#### Scenario: Company detail shows counts and timeline

- **WHEN** a user opens a company's detail view
- **THEN** the profile section SHALL display the company fields and the contact/negocio counts
- **AND** the detail view SHALL list the company's associated negocios
- **AND** the activity timeline SHALL display the company's activities

### Requirement: Deal detail view in Spanish

The system SHALL display a deal detail view showing the deal's profile fields, its current etapa, contact and company references, and an activity timeline, all in Spanish.

#### Scenario: Deal detail shows stage and timeline

- **WHEN** a user opens a deal's detail view
- **THEN** the profile section SHALL display the deal's fields and its current etapa
- **AND** the activity timeline SHALL display the deal's activities, including system activities for stage changes

### Requirement: Entity tag picker on detail views

The system SHALL provide a tag picker on contact, company, and deal detail views that lists the entity's current tags and allows attaching and detaching tags via `POST /api/crm/etiquetas/entity/:entityType/:entityId` and `DELETE /api/crm/etiquetas/entity/:entityType/:entityId/:tagId`.

#### Scenario: Attach a tag from the detail view

- **WHEN** a user opens a contact's detail view and attaches an existing tag
- **THEN** the tag SHALL be attached via `POST /api/crm/etiquetas/entity/contact/:contactId`
- **AND** the tag SHALL appear in the entity's tag list

#### Scenario: Detach a tag from the detail view

- **WHEN** a user removes a tag from a deal's detail view
- **THEN** the tag SHALL be detached via `DELETE /api/crm/etiquetas/entity/deal/:dealId/:tagId`
- **AND** the tag SHALL disappear from the entity's tag list

### Requirement: Activity type filter control in Spanish

The system SHALL provide a filter control on the Actividad view that filters the activity list by type. Options SHALL be: Todos, Nota, Llamada, Correo, Reunión, Tarea.

#### Scenario: Filter activities by type

- **WHEN** a user selects "Llamada" in the activity filter
- **THEN** the activity list SHALL refresh and SHALL display only activities of type call
- **AND** the filter SHALL be applied via the `tipo` parameter

### Requirement: Task activity form fields in Spanish

The system SHALL extend the activity creation form so that when the type Tarea is selected, the form SHALL collect `fecha_vencimiento` (due date) and `estado` (pendiente/hecha) fields in addition to asunto and contenido.

#### Scenario: Create a task activity with due date and estado

- **WHEN** a user selects Tarea, fills asunto, contenido, fecha_vencimiento, and estado, and submits
- **THEN** the activity SHALL be created with the task type and the due date and estado persisted

#### Scenario: Non-task activity hides task fields

- **WHEN** a user selects Nota or Llamada in the activity form
- **THEN** the fecha_vencimiento and estado fields SHALL NOT be displayed

### Requirement: Etiquetas tab shows Enterprise upgrade preview

The system SHALL display the Etiquetas tab grayed out with "Desbloquear con Enterprise" when the `crm_tags` feature is not enabled for the organization.

#### Scenario: Starter or Pro tier sees Enterprise gate

- **WHEN** a user without the `crm_tags` feature views the CRM
- **THEN** the Etiquetas tab SHALL be grayed out and SHALL display "Desbloquear con Enterprise"
- **AND** the system SHALL display an upgrade preview linking to billing

#### Scenario: Enterprise tier sees active Etiquetas tab

- **WHEN** a user with the `crm_tags` feature views the CRM
- **THEN** the Etiquetas tab SHALL be active and the tag manager SHALL be displayed
