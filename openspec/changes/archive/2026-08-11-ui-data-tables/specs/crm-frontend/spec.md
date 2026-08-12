# Delta Spec: crm-frontend — ui-data-tables

## ADDED Requirements

### Requirement: CRM and ticket lists use skeletons and distinct empty states

CRM and ticket list views SHALL render `Skeleton` rows while loading instead of "Cargando..." text, SHALL render a no-results state distinct from the empty-data state when filters match nothing, and SHALL keep the existing empty-data state for truly empty lists.

#### Scenario: Loading shows skeleton rows

- **WHEN** a CRM or ticket list query is pending
- **THEN** the view SHALL render skeleton rows

#### Scenario: Filter with no matches shows no-results state

- **WHEN** the list has data but the active search/filter matches nothing
- **THEN** the view SHALL display a no-results message with a clear-filter action, distinct from the empty-data message

#### Scenario: Truly empty list shows empty-data state

- **WHEN** the org has no rows
- **THEN** the view SHALL display the existing empty-data message
