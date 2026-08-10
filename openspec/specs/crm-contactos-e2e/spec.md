## Purpose

Define the E2E behavior of the CRM contacts module: full CRUD, search and lead-status filtering, and creation validation through the browser UI.

## Requirements

### Requirement: Contacts CRUD via browser UI

The system SHALL support full CRUD operations for contacts through the browser UI, verified by E2E tests.

#### Scenario: Create a contact
- **WHEN** a user navigates to the CRM Contactos tab
- **AND** clicks "Nuevo Contacto"
- **AND** fills in phone, name, email, and lead status
- **AND** submits the form
- **THEN** the new contact SHALL appear in the contacts table

#### Scenario: View a contact
- **WHEN** a user clicks a contact row in the table
- **THEN** the contact detail view SHALL display all fields

#### Scenario: Update a contact
- **WHEN** a user edits an existing contact
- **AND** changes the display name and lead status
- **THEN** the table SHALL reflect the updated values

#### Scenario: Delete a contact
- **WHEN** a user with admin role deletes a contact
- **THEN** the contact SHALL be removed from the table
- **AND** a success message SHALL appear

### Requirement: Contact search and filtering

The system SHALL support search and filter functionality for contacts, verified by E2E tests.

#### Scenario: Search contacts by name
- **WHEN** a user types a query in the search bar
- **THEN** the table SHALL display only matching contacts

#### Scenario: Filter contacts by lead status
- **WHEN** a user selects a lead status filter
- **THEN** the table SHALL display only contacts with that status

### Requirement: Contact validation

The system SHALL enforce validation rules for contact creation, verified by E2E tests.

#### Scenario: Duplicate phone number rejected
- **WHEN** a user creates a contact with a phone that already exists
- **THEN** an error message SHALL be displayed

#### Scenario: Empty phone number rejected
- **WHEN** a user submits the form without a phone number
- **THEN** a validation error SHALL be shown
