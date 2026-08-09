## ADDED Requirements

### Requirement: CRM page is accessible from sidebar navigation

The system SHALL add a "CRM" navigation item to the dashboard sidebar with paywall check (hidden if no active subscription) and `contact:view` permission check.

#### Scenario: User with active subscription sees CRM in sidebar

- **WHEN** a user with an active subscription and `contact:view` permission logs into the dashboard
- **THEN** the sidebar SHALL display a "CRM" navigation item
- **AND** clicking it SHALL navigate to `/dashboard/crm`

#### Scenario: Free tier user does not see CRM

- **WHEN** a user without an active subscription logs in
- **THEN** the sidebar SHALL NOT display the CRM navigation item

### Requirement: CRM dashboard is an SPA with view navigation in Spanish

The system SHALL implement the CRM page as a single-page application using URL search params for view state. All labels, buttons, and headers SHALL be in Colombian Spanish.

#### Scenario: Default view shows contact list with Spanish UI

- **WHEN** a user navigates to `/dashboard/crm`
- **THEN** the view SHALL default to showing the contact list ("Contactos")
- **AND** the URL SHALL sync to `?view=contactos`
- **AND** all column headers SHALL be in Spanish: Nombre, Teléfono, Correo, Empresa, Estado, Último Contacto

#### Scenario: Navigation between views

- **WHEN** a user navigates from contactos to negocios view
- **THEN** the URL SHALL update to `?view=negocios`
- **AND** the deals kanban SHALL be displayed with Spanish stage names

### Requirement: Tab bar builds dynamically from enabled features

The system SHALL build the CRM navigation tab bar from the entitlement's enabled features. Unavailable features SHALL show as disabled tabs with upgrade previews.

#### Scenario: Pro tier sees all available tabs

- **WHEN** a Pro tier user views the CRM
- **THEN** the tab bar SHALL show: Contactos, Empresas, Negocios, Actividad
- **AND** Etiquetas SHALL be grayed out with "Desbloquear con Enterprise"

#### Scenario: Starter tier sees only contactos

- **WHEN** a Starter tier user views the CRM
- **THEN** the tab bar SHALL show: Contactos (active)
- **AND** Empresas, Negocios SHALL be grayed out with "Desbloquear con Pro"

### Requirement: Contact list with Spanish labels and Colombian-relevant columns

The system SHALL display a data table of contacts with Spanish column headers, search input, and filters. Columns SHALL include: Nombre, Teléfono, Correo, Documento, Empresa, Estado, Último Contacto.

#### Scenario: Contact list loads with Spanish UI

- **WHEN** the contact list view loads
- **THEN** the system SHALL display columns in Spanish
- **AND** the search placeholder SHALL read "Buscar contactos..."

### Requirement: Contact detail with activity timeline and deals

The system SHALL display a contact detail view showing profile fields (including Tipo Documento, Número Documento), associated negocios, and an activity timeline, all in Spanish.

#### Scenario: Contact detail shows documento fields

- **WHEN** a user clicks a contact from the list
- **THEN** the system SHALL display Tipo Documento and Número Documento in the contact profile section
- **AND** action buttons SHALL read "Editar", "Agregar nota", "Crear negocio"

### Requirement: Deal kanban board in Colombian Spanish

The system SHALL display negocios as cards arranged in pipeline etapa columns. Column headers SHALL use Spanish stage names. Dropdown labels SHALL be in Spanish. The kanban SHALL show a pipeline selector defaulting to the `es_predeterminado` pipeline ("Pipeline de Ventas") and SHALL allow moving a negocio between etapas by dragging the card onto a stage column, persisting the move via `PUT /api/crm/negocios/:id/etapa` with the source and target stage names.

#### Scenario: Kanban shows deals with Spanish UI

- **WHEN** the negocios view loads
- **THEN** etapa columns SHALL show Spanish names (Prospección, Calificado, Propuesta, etc.)
- **AND** deal cards SHALL show monto formatted as "$10.000.000 COP"
- **AND** the pipeline selector SHALL read "Pipeline de Ventas"

#### Scenario: Deal is moved by dragging to another stage column

- **WHEN** a user drags a deal card from one etapa column onto another etapa column and releases it
- **THEN** the system SHALL call `PUT /api/crm/negocios/:id/etapa` with the target `stage_id` and the source/target stage names
- **AND** the deal SHALL appear in the target column after the list refreshes

### Requirement: Activity creation form in Spanish

The system SHALL provide a form in Spanish to create activities. Type options SHALL be: Nota, Llamada, Correo, Reunión, Tarea.

#### Scenario: User logs a call in Spanish

- **WHEN** a user fills in the activity form with tipo='Llamada', asunto, and contenido
- **THEN** the activity SHALL be created with type 'call' (stored in English, displayed in Spanish)
- **AND** the actividad timeline SHALL refresh

### Requirement: Pipeline editor with Spanish labels

The system SHALL provide a pipeline editor view at `?view=pipelines` (gated by the `crm_deals` feature) where pipeline owners can view pipelines, create a pipeline with its initial etapas, and edit a stage's nombre, color, or probabilidad. Labels and buttons SHALL be in Spanish ("Nuevo pipeline", "Agregar Etapa", "Guardar", "Cancelar"). The stage list SHALL render "Salida" when a stage's probabilidad is null.

#### Scenario: Editing a stage's details in Spanish

- **WHEN** a user edits a stage's nombre, color, or probabilidad and saves
- **THEN** the stage SHALL be updated via `PUT /api/crm/pipelines/:id/etapas/:stageId`
- **AND** buttons SHALL read "Guardar", "Cancelar", "Agregar Etapa"

#### Scenario: Stage with null probability renders as Salida

- **WHEN** a stage with `probabilidad = null` is displayed in the editor
- **THEN** the stage SHALL render the label "Salida" instead of a percentage

### Requirement: Company list with Colombian-relevant columns

The system SHALL display companies with Spanish column headers: Nombre, NIT, Sector, Ciudad, Tipo, Contactos, Negocios.

#### Scenario: Company list shows Colombian data

- **WHEN** a user views the company list
- **THEN** columns SHALL include NIT and Ciudad
- **AND** the search placeholder SHALL read "Buscar empresas por nombre, NIT o sector..."

### Requirement: Upgrade preview banners for unavailable features

The system SHALL display upgrade CTAs for features not in the current plan but available in higher tiers.

#### Scenario: Starter user sees upgrade prompt for Companies

- **WHEN** a Starter user views the CRM and clicks the disabled "Empresas" tab
- **THEN** the system SHALL display: "Empresas es una funcionalidad Pro. Actualiza tu plan para gestionar empresas."
- **AND** SHALL show an "Actualizar a Pro" button linking to billing

### Requirement: Mutation errors display in Spanish

The system SHALL display error toasts in Spanish after failed mutations.

#### Scenario: Create contact with duplicate email

- **WHEN** a user creates a contact with an email that already exists
- **THEN** a toast SHALL display: "Ya existe un contacto con este correo electrónico."

### Requirement: All views lazy-load data based on feature availability

The system SHALL use TanStack Query with `enabled` options gated by both the active view AND the feature flag.

#### Scenario: Negocios data not fetched when feature disabled

- **WHEN** a Starter user (no crm_deals) is on the CRM page
- **THEN** the negocios query SHALL be disabled regardless of active view
- **AND** no `/api/crm/negocios` requests SHALL be made

### Requirement: Contact creation form in Spanish

The system SHALL provide a "Nuevo contacto" button in the Contactos view that opens a create dialog with fields: phone (`phone`, required), name (`display_name`), email (`email`), and commercial status (`lead_status`). Labels, buttons, and validation messages SHALL be in Colombian Spanish.

#### Scenario: Create contact from dialog

- **WHEN** a user clicks "Nuevo contacto" and submits a valid phone number and optional name/email/lead_status
- **THEN** a new contact SHALL be created via `POST /api/crm/contactos`
- **AND** the contact list SHALL refresh and SHALL display the new row

#### Scenario: Empty phone shows Spanish validation error

- **WHEN** a user submits the contact form without a phone number
- **THEN** the system SHALL display a Spanish validation error (e.g. "El teléfono es requerido") without calling the API
- **AND** the form SHALL remain open

### Requirement: Contact edit form in Spanish

The system SHALL provide an "Editar" action on each contact row that opens the contact dialog pre-filled with the current values and saves via `PUT /api/crm/contactos/:id`.

#### Scenario: Edit contact display name

- **WHEN** a user clicks "Editar" on a contact row, changes `display_name`, and clicks "Guardar"
- **THEN** the contact SHALL be updated via `PUT /api/crm/contactos/:id`
- **AND** the updated name SHALL be visible in the row

### Requirement: Contact delete with confirmation

The system SHALL provide an "Eliminar" action on each contact row that asks for confirmation before deleting via `DELETE /api/crm/contactos/:id`.

#### Scenario: Delete contact after confirmation

- **WHEN** a user clicks "Eliminar" on a contact row and confirms the dialog
- **THEN** the contact SHALL be deleted via `DELETE /api/crm/contactos/:id`
- **AND** the row SHALL disappear from the list
- **AND** a success toast SHALL be displayed

### Requirement: Company create form in Spanish

The system SHALL provide a "Nueva empresa" button in the Empresas view that opens a create dialog with fields: name (`name`, required), NIT (`nit`), sector (`sector`), and city (`ciudad`). Labels and buttons SHALL be in Colombian Spanish.

#### Scenario: Create company from dialog

- **WHEN** a user clicks "Nueva empresa" and submits a valid name
- **THEN** a new company SHALL be created via `POST /api/crm/empresas`
- **AND** the company list SHALL refresh and SHALL display the new row

#### Scenario: Duplicate company name shows error

- **WHEN** a user creates a company whose name already exists in the organization
- **THEN** the system SHALL display the error "Ya existe una empresa con este nombre."
- **AND** the create dialog SHALL remain open

### Requirement: Company edit form in Spanish

The system SHALL provide an "Editar" action on each company row that opens the company dialog pre-filled with the current values and saves via `PUT /api/crm/empresas/:id`.

#### Scenario: Edit company name

- **WHEN** a user clicks "Editar" on a company row, changes `name`, and clicks "Guardar"
- **THEN** the company SHALL be updated via `PUT /api/crm/empresas/:id`
- **AND** the updated name SHALL be visible in the row

### Requirement: Company delete with confirmation

The system SHALL provide an "Eliminar" action on each company row that asks for confirmation before deleting via `DELETE /api/crm/empresas/:id`.

#### Scenario: Delete company after confirmation

- **WHEN** a user clicks "Eliminar" on a company row and confirms the dialog
- **THEN** the company SHALL be deleted via `DELETE /api/crm/empresas/:id`
- **AND** the row SHALL disappear from the list
- **AND** a success toast SHALL be displayed

### Requirement: Mutation errors display in Spanish toasts

The system SHALL display mutation failures as Spanish error toasts using the backend-provided conflict messages when available.

#### Scenario: Duplicate contact email shows Spanish toast

- **WHEN** a contact is created or updated with an email that already exists in the organization
- **THEN** a toast SHALL display "Ya existe un contacto con este correo electrónico."

#### Scenario: Generic mutation failure shows Spanish toast

- **WHEN** a contact or company mutation fails for a non-conflict reason
- **THEN** a toast SHALL display a Spanish error message (e.g. "Solicitud inválida" or "Error de conexión")

### Requirement: CRUD buttons gated by entitlement features

The system SHALL show contact and company CRUD buttons only when the corresponding entitlement feature is enabled for the organization.

#### Scenario: Company CRUD hidden for Starter tier

- **WHEN** a Starter user (no `crm_companies` feature) views the Empresas tab
- **THEN** the "Nueva empresa" button SHALL NOT be displayed

#### Scenario: Contact CRUD shown when feature enabled

- **WHEN** an organization has contact management enabled
- **THEN** the "Nuevo contacto" button SHALL be displayed in the Contactos view

### Requirement: Deal creation form in Colombian Spanish

The system SHALL provide a "Nuevo negocio" button in the Negocios view that opens a create dialog with fields: `nombre` (required), `monto` (optional), `moneda` (default "COP"), an optional `empresa` select, an optional `contacto` select, and an optional `etapa` select populated from the active pipeline's stages. Labels, buttons, and validation messages SHALL be in Colombian Spanish.

#### Scenario: Create deal from kanban

- **WHEN** a user clicks "Nuevo negocio", fills the form, and submits
- **THEN** the deal SHALL be created via `POST /api/crm/negocios` with the selected pipeline_id
- **AND** the kanban SHALL refresh and SHALL display the new card in the corresponding etapa column

#### Scenario: Empty nombre shows Spanish validation error

- **WHEN** a user submits the deal form without a nombre
- **THEN** the system SHALL display a Spanish validation error without calling the API

### Requirement: Deal edit and delete from the kanban

The system SHALL provide per-card edit and delete actions. Editing opens the deal dialog prefilled with the deal's data and SHALL persist via `PUT /api/crm/negocios/:id`. Deleting SHALL require confirmation via a Spanish confirmation dialog and SHALL delete via `DELETE /api/crm/negocios/:id`.

#### Scenario: Deal edited from card menu

- **WHEN** a user opens a deal card's menu, selects "Editar", changes the monto, and saves
- **THEN** the deal SHALL be updated via `PUT /api/crm/negocios/:id`
- **AND** the kanban SHALL refresh and SHALL show the updated amount

#### Scenario: Deal deleted with confirmation

- **WHEN** a user opens a deal card's menu, selects "Eliminar", and confirms in the Spanish dialog
- **THEN** the deal SHALL be deleted via `DELETE /api/crm/negocios/:id`
- **AND** the card SHALL disappear from the kanban

### Requirement: Pipeline creation with stages in one dialog flow

The system SHALL provide a "Nuevo pipeline" dialog in the Pipelines view that collects the pipeline name and one or more stage rows (nombre, color). On submit, the system SHALL create the pipeline via `POST /api/crm/pipelines` and then create each stage via `POST /api/crm/pipelines/:id/etapas`.

#### Scenario: Create pipeline with stages

- **WHEN** a user clicks "Nuevo pipeline", enters a name, adds two stages with "Agregar Etapa", and submits
- **THEN** the pipeline SHALL be created via `POST /api/crm/pipelines`
- **AND** each stage SHALL be created via `POST /api/crm/pipelines/:id/etapas`
- **AND** the pipeline SHALL appear in the pipeline list with its stages

## ADDED Requirements

### Requirement: Auth pages use pre-built Stytch components

The system SHALL replace any custom auth-related pages (`/login`, `/signup`, `/settings/members`, `/settings/sso`) with Stytch pre-built B2B components from `@stytch/nextjs/b2b`.

#### Scenario: Login page is the Stytch component

- **WHEN** a user navigates to `/login`
- **THEN** the page SHALL render `<StytchB2B />` with Discovery flow
- **AND** NO custom form components SHALL exist on the page

#### Scenario: Settings page renders Stytch admin portal

- **WHEN** an authenticated admin navigates to `/settings`
- **THEN** the page SHALL render `<AdminPortalMemberManagement />` and `<AdminPortalSSO />`
- **AND** NO custom member management or SSO form components SHALL exist on the page

### Requirement: Protected route gating via edge middleware

The system SHALL gate all protected frontend routes (`/dashboard/:path*`, `/settings/:path*`) through the edge middleware instead of client-side or server-side per-page checks.

#### Scenario: Unauthenticated user redirected from dashboard

- **WHEN** an unauthenticated user navigates to `/dashboard/crm`
- **THEN** the edge middleware SHALL redirect to `/login`
- **AND** no CRM page code SHALL execute

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
