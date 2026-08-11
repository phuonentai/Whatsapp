## MODIFIED Requirements

### Requirement: API envelope carries optional total on paginated lists

The frontend API envelope type SHALL support an optional `total` field alongside `data` on paginated CRM list responses, so list views can render page controls with accurate counts. Non-paginated consumers SHALL be unaffected.

#### Scenario: List hook exposes total count

- **WHEN** a paginated CRM list request resolves with a `total` field
- **THEN** the envelope SHALL carry `total` as a number alongside the `data` array
