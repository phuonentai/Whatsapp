## ADDED Requirements

### Requirement: CRM view export buttons

Each of the Contactos, Empresas, Negocios, and Actividades list views SHALL provide a Spanish-language export action that downloads the corresponding CSV via the bulk-export endpoints. The download SHALL use a fetch-and-blob flow carrying the Stytch session token in the request headers; a bare `window.location` navigation SHALL NOT be used because it cannot attach the session token. The action SHALL be hidden when the user lacks the relevant `export` permission.

#### Scenario: Contact list exports CSV

- **WHEN** a user with `contact:export` clicks the export action on the Contactos view
- **THEN** the frontend SHALL fetch `GET /api/crm/export/contactos.csv` with the session token
- **AND** SHALL trigger a browser download of the CSV content

#### Scenario: Export action hidden without permission

- **WHEN** the current user lacks the `export` permission for a view's resource
- **THEN** the export action SHALL NOT be rendered

### Requirement: Contact import modal

The Contactos view SHALL provide an import action (visible with `contact:manage`) that opens a modal with a downloadable template link (`GET /api/crm/import/contactos/template.csv`), a CSV file picker, and a result summary showing imported count, omitted count, and per-row errors after submission to `POST /api/crm/import/contactos`. User-facing strings SHALL be in Colombian Spanish.

#### Scenario: Template downloads from the modal

- **WHEN** a user opens the import modal and clicks the template link
- **THEN** the frontend SHALL download the template CSV with the expected columns and example rows

#### Scenario: Upload shows a result summary

- **WHEN** a user submits a CSV through the modal
- **THEN** the modal SHALL display the imported, omitted, and error counts returned by the import endpoint
- **AND** per-row errors SHALL be listed with their row numbers
