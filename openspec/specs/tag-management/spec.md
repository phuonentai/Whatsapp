## Purpose

Defines the tag entity with organization scoping and attach and detach operations across contacts, companies, and deals.

## Requirements

### Requirement: Tag entity with organization scoping

The system SHALL store tags in `crm.tags` scoped by `organization_id`, with a unique constraint on `(organization_id, name)` and an optional `color` field.

#### Scenario: Tag created for an organization

- **WHEN** a tag is created with name and color for an organization
- **THEN** the tag SHALL be persisted and scoped to that organization

#### Scenario: Duplicate tag name in same organization is rejected

- **WHEN** a tag is created with a name that already exists for the organization
- **THEN** the system SHALL return a conflict error

### Requirement: Tags can be attached to contacts, companies, and deals

The system SHALL use the `crm.entity_tags` junction table to associate tags with entities, supporting `entity_type` values of `contact`, `company`, and `deal`.

#### Scenario: Tag attached to a contact

- **WHEN** a tag is attached to a contact via `POST /api/crm/contact/:id/tags`
- **THEN** the association SHALL be created in entity_tags
- **AND** the contact's tags SHALL include this tag

#### Scenario: Tag attached to a deal

- **WHEN** a tag is attached to a deal via `POST /api/crm/deal/:id/tags`
- **THEN** the association SHALL be persisted

#### Scenario: Same tag attached twice to same entity is rejected

- **WHEN** a tag is attached to an entity that already has that tag
- **THEN** the system SHALL return a conflict error
- **AND** a duplicate row SHALL NOT be created

### Requirement: Tags can be detached from entities

The system SHALL support removing a tag from an entity via `DELETE /api/crm/:entityType/:entityId/tags/:tagId`.

#### Scenario: Tag removed from a contact

- **WHEN** a tag is detached from a contact
- **THEN** the entity_tags row SHALL be deleted
- **AND** the contact's tag list SHALL NOT include the removed tag

#### Scenario: Detaching non-existent tag returns success

- **WHEN** a tag that is not attached to the entity is detached
- **THEN** the system SHALL return success (idempotent)

### Requirement: Entity responses include tags

The system SHALL include an array of tag objects when returning contact, company, or deal details.

#### Scenario: Contact detail includes tags

- **WHEN** a GET request is made to `/api/crm/contacts/1` and the contact has tags
- **THEN** the response SHALL include a `tags` array with tag id, name, and color

#### Scenario: Deal detail includes tags

- **WHEN** a GET request is made to `/api/crm/deals/1`
- **THEN** the response SHALL include associated tags

### Requirement: Tag list is paginated

The system SHALL provide a paginated list of tags for the organization via `GET /api/crm/tags`.

#### Scenario: List all tags for organization

- **WHEN** a GET request is made to `/api/crm/tags`
- **THEN** the system SHALL return all tags for the organization with pagination

### Requirement: Tag deletion cascades to entity_tags

The system SHALL use `ON DELETE CASCADE` for the tag-to-entity_tags foreign key.

#### Scenario: Tag deleted while attached to entities

- **WHEN** a tag that is attached to contacts and deals is deleted
- **THEN** the tag SHALL be deleted
- **AND** all associated entity_tags rows SHALL be automatically deleted

### Requirement: Tag management is RBAC-protected

The system SHALL require `contact:manage` permission for tag create, update, and delete operations.

#### Scenario: User with manage permission creates a tag

- **WHEN** a user with `contact:manage` permission creates a tag
- **THEN** the tag SHALL be created successfully

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
