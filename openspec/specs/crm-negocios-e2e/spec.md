## Purpose

Define the E2E behavior of the CRM deals (Negocios) module: full CRUD on the Kanban board, pipeline stage movement, won/lost status changes, and creation linked to contacts and companies.

## Requirements

### Requirement: Deals Kanban CRUD via browser UI

The system SHALL support full CRUD operations for deals through the Kanban board UI, verified by E2E tests.

#### Scenario: Create a deal
- **WHEN** a user navigates to the Negocios tab
- **AND** clicks "Nuevo Negocio"
- **AND** fills in name, amount, expected close date
- **AND** submits the form
- **THEN** the new deal SHALL appear as a card in the first pipeline stage column

#### Scenario: View a deal
- **WHEN** a user clicks a deal card on the Kanban board
- **THEN** the deal detail SHALL display all fields including linked contact and company names

#### Scenario: Update a deal
- **WHEN** a user edits an existing deal
- **AND** changes the amount and probability
- **THEN** the deal card SHALL reflect the updated values

#### Scenario: Delete a deal
- **WHEN** a user deletes a deal
- **THEN** the deal card SHALL be removed from the Kanban board

### Requirement: Deal stage movement

The system SHALL support moving deals between pipeline stages, verified by E2E tests.

#### Scenario: Move deal to next stage
- **WHEN** a user drags a deal card from one stage column to the next
- **THEN** the deal SHALL appear in the target stage column
- **AND** the stage name SHALL be updated in the deal detail

#### Scenario: Change deal status to won
- **WHEN** a user marks a deal as "Ganado" (won)
- **THEN** the deal SHALL move to the "Cerrado Ganado" stage (or designated won stage)
- **AND** the deal status SHALL be updated

### Requirement: Deal creation with linked entities

The system SHALL support creating deals linked to contacts and companies, verified by E2E tests.

#### Scenario: Create deal linked to contact and company
- **WHEN** a user creates a deal
- **AND** selects an existing contact and company via the picker
- **THEN** the deal SHALL display the contact name and company name in its detail
