## ADDED Requirements

### Requirement: Entity tag list read endpoint

The system SHALL provide `GET /api/crm/etiquetas/entity/:entityType/:entityId` returning all tags attached to the given entity.

#### Scenario: List tags attached to a contact

- **WHEN** a GET request is made to `/api/crm/etiquetas/entity/contact/1`
- **THEN** the system SHALL return all tags attached to contact ID 1 with their id, name, and color

#### Scenario: Entity with no tags returns empty list

- **WHEN** a GET request is made to `/api/crm/etiquetas/entity/company/5` and the company has no tags
- **THEN** the system SHALL return an empty array

### Requirement: Tag can be updated

The system SHALL support updating a tag's name and color via `PUT /api/crm/etiquetas/:id`.

#### Scenario: Rename and recolor a tag

- **WHEN** a PUT request is made to `/api/crm/etiquetas/1` with a new name and color
- **THEN** the tag SHALL be updated with the new values

#### Scenario: Update to duplicate tag name is rejected

- **WHEN** a tag is updated to a name that already exists for the organization
- **THEN** the system SHALL return a conflict error
