## Purpose

Define the E2E behavior of the CRM companies module: full CRUD, search by name or NIT, and duplicate-name validation through the browser UI.

## Requirements

### Requirement: Companies CRUD via browser UI

The system SHALL support full CRUD operations for companies through the browser UI, verified by E2E tests.

#### Scenario: Create a company
- **WHEN** a user navigates to the Empresas tab
- **AND** fills in name, NIT, sector, ciudad, and phone
- **AND** submits the form
- **THEN** the new company SHALL appear in the companies table

#### Scenario: View a company
- **WHEN** a user clicks a company row
- **THEN** the company detail SHALL display all fields and associated contact/deal counts

#### Scenario: Update a company
- **WHEN** a user edits an existing company
- **AND** changes the name and sector
- **THEN** the table SHALL reflect the updated values

#### Scenario: Delete a company
- **WHEN** a user deletes a company
- **THEN** the company SHALL be removed from the table

### Requirement: Company search

The system SHALL support search functionality for companies, verified by E2E tests.

#### Scenario: Search companies by name or NIT
- **WHEN** a user types a query in the search bar
- **THEN** the table SHALL display only matching companies

### Requirement: Company validation

The system SHALL enforce validation rules for company creation, verified by E2E tests.

#### Scenario: Duplicate company name rejected
- **WHEN** a user creates a company with a name that already exists in the org
- **THEN** an error message SHALL be displayed
