## ADDED Requirements

### Requirement: Deals Kanban CRUD via browser UI

The E2E tests SHALL verify full CRUD operations for deals through the Kanban board UI.

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

The E2E tests SHALL verify that deals can move between pipeline stages.

#### Scenario: Move deal to next stage
- **WHEN** a user drags a deal card from one stage column to the next
- **THEN** the deal SHALL appear in the target stage column
- **AND** the stage name SHALL be updated in the deal detail

#### Scenario: Change deal status to won
- **WHEN** a user marks a deal as "Ganado" (won)
- **THEN** the deal SHALL move to the "Cerrado Ganado" stage (or designated won stage)
- **AND** the deal status SHALL be updated

### Requirement: Deal creation with linked entities

The E2E tests SHALL verify creating deals linked to contacts and companies.

#### Scenario: Create deal linked to contact and company
- **WHEN** a user creates a deal
- **AND** selects an existing contact and company via the picker
- **THEN** the deal SHALL display the contact name and company name in its detail
