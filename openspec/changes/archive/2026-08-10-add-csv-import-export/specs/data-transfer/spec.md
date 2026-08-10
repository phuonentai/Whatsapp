## ADDED Requirements

### Requirement: CSV export of CRM entities

The system SHALL provide bulk CSV export endpoints for the four main CRM entities, gated by the RBAC `export` action on each resource: `GET /api/crm/export/contactos.csv` (resource `contact`), `GET /api/crm/export/empresas.csv` (resource `contact`), `GET /api/crm/export/negocios.csv` (resource `deal`), and `GET /api/crm/export/actividades.csv` (resource `activity`). Export SHALL be scoped to the authenticated organization resolved from the request context, SHALL NOT trust any user-supplied organization identifier, and SHALL stream rows to the response as `text/csv` with `Content-Disposition: attachment` and a UTF-8 byte order mark (BOM) prefix so Spanish accents render correctly in Microsoft Excel.

#### Scenario: Export contacts as CSV

- **WHEN** a user with the `contact:export` permission requests `GET /api/crm/export/contactos.csv`
- **THEN** the system SHALL respond `200` with `Content-Type: text/csv`, a UTF-8 BOM prefix, and one header row plus one row per contact in the requesting organization
- **AND** the header row SHALL use Spanish column names matching the CRM list view

#### Scenario: Export denied without export permission

- **WHEN** a user without the `contact:export` permission requests a bulk export endpoint
- **THEN** the system SHALL return HTTP 403 with no data

#### Scenario: Export is organization-scoped

- **WHEN** a user of organization A requests an export
- **THEN** the response SHALL contain only organization A's records, regardless of any identifiers in query parameters or body

#### Scenario: Export streams without loading the full table

- **WHEN** an organization has more records than a single page
- **THEN** the system SHALL paginate through the repository and emit rows incrementally without buffering the entire dataset in memory

### Requirement: CSV formula injection sanitization

The system SHALL sanitize CSV cell values on export so that cells beginning with `=`, `+`, `-`, or `@` cannot execute as spreadsheet formulas when opened in Excel. Each such value SHALL be prefixed with a single quote (`'`) before being written, or otherwise escaped so the spreadsheet renders it as literal text.

#### Scenario: Formula-prefixed value is neutralized

- **WHEN** a field value starts with `=` (e.g., `=HYPERLINK(...)`)
- **THEN** the exported cell SHALL begin with a single quote so the spreadsheet treats it as text, not a formula

### Requirement: Withdrawn-consent PII masking in export

The system SHALL mask personally identifiable information for contacts whose `consent_status` is `withdrawn` in every CSV export, consistent with the Habeas Data export invariant: phone, name, email, document type, and document number SHALL be replaced with placeholders (`[TELEFONO]`, `[NOMBRE]`, `[EMAIL]`, `[DOCUMENTO]`), regardless of who performs the export.

#### Scenario: Withdrawn contact exported masked

- **WHEN** an export includes a contact with `consent_status = 'withdrawn'`
- **THEN** that contact's phone, name, email, and document fields SHALL contain placeholder tokens

#### Scenario: Granted contact exported with real PII

- **WHEN** an export includes a contact with `consent_status = 'granted'`
- **THEN** that contact's PII fields SHALL be exported as stored

### Requirement: Export audit logging

The system SHALL record an audit event for every bulk export containing the acting member, the organization, the exported entity, the row count, and the timestamp.

#### Scenario: Export produces an audit entry

- **WHEN** a bulk export completes successfully
- **THEN** the system SHALL persist an audit event with member id, organization id, entity, row count, and timestamp

### Requirement: CSV contact import from strict template

The system SHALL provide `GET /api/crm/import/contactos/template.csv` (gated `contact:view`) returning a CSV template with the exact import columns and example rows, and `POST /api/crm/import/contactos` (gated `contact:manage`) accepting a CSV upload. Import SHALL validate each row against the template: required columns `teléfono` and `nombre`; optional `email`, `tipo_documento`, `numero_documento`, `empresa`, `origen`, `estado`. Invalid document types SHALL be rejected per the existing contact validation rules (CC, NIT, CE, TI, PP). Import SHALL be organization-scoped from the request context.

#### Scenario: Valid template row imports a contact

- **WHEN** a user with `contact:manage` uploads a CSV with a row containing a valid phone and name
- **THEN** the system SHALL create the contact in the requesting organization
- **AND** the response SHALL report the row as imported

#### Scenario: Invalid row is reported, not imported

- **WHEN** an uploaded CSV contains a row with an invalid phone or missing required column
- **THEN** the system SHALL NOT import that row
- **AND** the response SHALL include a row-level error with the row number and reason

#### Scenario: Import is organization-scoped

- **WHEN** a user of organization A uploads a CSV
- **THEN** all imported contacts SHALL be created under organization A only

### Requirement: Import deduplication by phone

The system SHALL deduplicate contact imports by the existing unique constraint `(organization_id, phone_number)`: when a row's phone already exists for the organization, the row SHALL be skipped and reported as omitted. The existing contact SHALL NOT be overwritten by import.

#### Scenario: Duplicate phone is skipped

- **WHEN** an uploaded CSV contains a phone already present in the organization
- **THEN** the system SHALL NOT create or update the existing contact
- **AND** the response SHALL report the row as omitted

### Requirement: Import limits and content validation

The system SHALL enforce hard limits on contact import: a maximum of 5000 rows, a maximum upload size, and content sniffing that rejects non-CSV payloads before parsing. Import SHALL return a summary response `{importados, omitidos, errores:[{fila, razon}]}` after processing.

#### Scenario: Import over row cap is rejected

- **WHEN** an uploaded CSV exceeds 5000 rows
- **THEN** the system SHALL reject the import with an error before writing any rows

#### Scenario: Non-CSV payload is rejected

- **WHEN** an upload's content does not match CSV
- **THEN** the system SHALL reject the request with HTTP 400

#### Scenario: Import returns a summary

- **WHEN** a CSV import completes
- **THEN** the system SHALL return counts of imported rows, omitted rows, and a list of per-row errors
