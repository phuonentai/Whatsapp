## ADDED Requirements

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
