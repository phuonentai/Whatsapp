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

The system SHALL display negocios as cards arranged in pipeline etapa columns. Column headers SHALL use Spanish stage names. Dropdown labels SHALL be in Spanish.

#### Scenario: Kanban shows deals with Spanish UI

- **WHEN** the negocios view loads
- **THEN** etapa columns SHALL show Spanish names (Prospección, Calificado, Propuesta, etc.)
- **AND** deal cards SHALL show monto formatted as "$10.000.000 COP"
- **AND** the pipeline selector SHALL read "Pipeline de Ventas"

### Requirement: Activity creation form in Spanish

The system SHALL provide a form in Spanish to create activities. Type options SHALL be: Nota, Llamada, Correo, Reunión, Tarea.

#### Scenario: User logs a call in Spanish

- **WHEN** a user fills in the activity form with tipo='Llamada', asunto, and contenido
- **THEN** the activity SHALL be created with type 'call' (stored in English, displayed in Spanish)
- **AND** the actividad timeline SHALL refresh

### Requirement: Pipeline editor with Spanish labels

The system SHALL provide a pipeline editor view where pipeline owners can view, create, edit, and reorder etapas. Labels SHALL be in Spanish.

#### Scenario: Editing a stage's details in Spanish

- **WHEN** a user edits a stage's nombre, color, or probabilidad and saves
- **THEN** the stage SHALL be updated
- **AND** buttons SHALL read "Guardar", "Cancelar", "Agregar Etapa"

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
