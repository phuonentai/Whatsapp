## ADDED Requirements

### Requirement: Deal exposes active quote and synced monto

The system SHALL expose on a deal its active quote (highest version, unless a newer `aprobada` exists — the `aprobada` quote is canonical) and SHALL sync the deal `monto` from the approved quote's total when a quote is approved.

#### Scenario: Approved quote updates deal amount
- **WHEN** a quote on a deal transitions to `aprobada`
- **THEN** the deal `monto` SHALL be set to the quote total
- **AND** the deal detail SHALL reflect the updated amount

#### Scenario: Deal detail shows active quote
- **WHEN** a member views a deal that has quotes
- **THEN** the deal detail SHALL show the active quote reference (number, version, status, total)
