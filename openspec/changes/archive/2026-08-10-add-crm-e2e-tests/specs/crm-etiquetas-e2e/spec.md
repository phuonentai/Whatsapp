## ADDED Requirements

### Requirement: Tag CRUD via browser UI

The E2E tests SHALL verify tag creation and deletion through the tag manager UI.

#### Scenario: Create a tag
- **WHEN** a user navigates to the Etiquetas tab
- **AND** clicks "Nueva Etiqueta"
- **AND** enters a tag name and selects a color
- **THEN** the new tag SHALL appear in the tag list

#### Scenario: Delete a tag
- **WHEN** a user deletes an existing tag
- **THEN** the tag SHALL be removed from the list

#### Scenario: Duplicate tag name rejected
- **WHEN** a user creates a tag with a name that already exists
- **THEN** an error message SHALL be displayed

### Requirement: Entity tagging via browser UI

The E2E tests SHALL verify tagging and untagging of CRM entities.

#### Scenario: Tag a contact
- **WHEN** a user views a contact detail
- **AND** selects a tag from the tag picker
- **THEN** the tag SHALL appear on the contact

#### Scenario: Tag a deal
- **WHEN** a user views a deal detail
- **AND** selects a tag from the tag picker
- **THEN** the tag SHALL appear on the deal

#### Scenario: Untag an entity
- **WHEN** a user removes a tag from a contact or deal
- **THEN** the tag SHALL be removed from the entity
