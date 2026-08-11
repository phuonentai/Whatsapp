# Delta Spec: crm-frontend — ux-error-recovery

## ADDED Requirements

### Requirement: CRM list views render error and retry states

Each CRM list view (Contactos, Empresas, Negocios, Actividad) and the ticket list SHALL render an inline error state with a Spanish retry action when its query fails, instead of an indefinite "Cargando..." indicator.

#### Scenario: Contacts query failure shows Spanish error with retry

- **WHEN** the contacts query fails
- **THEN** the Contactos view SHALL render an error state (e.g. "Error al cargar los contactos") with a "Reintentar" button
- **AND** clicking "Reintentar" SHALL re-run the query

#### Scenario: Ticket query failure shows error with retry

- **WHEN** the ticket list query fails
- **THEN** the ticket list SHALL render an error state with a retry action instead of an indefinite "Cargando..."

#### Scenario: Failed dialog mutation keeps dialog open

- **WHEN** a contact, company, or deal create/update mutation fails
- **THEN** a Spanish error toast SHALL be shown and the dialog SHALL remain open with the entered values intact
