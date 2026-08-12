## ADDED Requirements

### Requirement: Versioned quote entity linked to a deal

The system SHALL store quotes org-scoped in `crm.quotes` linked to a deal, with version numbers unique per `(organization_id, deal_id, version)`, a consecutive per-org document number (prefix from branding config, default `COT`), currency COP, denormalized totals, a branding snapshot key captured at creation, and an extensible `payload` JSONB. Line items SHALL live in `crm.quote_items` (position, description, optional SKU reference, quantity, unit price, discount percent, snapshotted IVA percent, computed line total) cascading with the quote. Quotes SHALL be creatable only for deals in the same organization; quote writes SHALL require `org:manage`, reads `org:view`.

#### Scenario: Create quote with line items
- **WHEN** an `org:manage` member creates a quote for a deal with line items and IVA
- **THEN** the quote SHALL be persisted with version 1, status `borrador`, number from the per-org sequence, branding snapshot key, and computed subtotal/IVA/total
- **AND** the line items SHALL be persisted with computed line totals

#### Scenario: Duplicate version rejected
- **WHEN** a second quote with the same `(organization_id, deal_id, version)` is attempted
- **THEN** the system SHALL reject the write
- **AND** SHALL NOT create a duplicate row

#### Scenario: Quote for another org's deal rejected
- **WHEN** a member attempts to create a quote for a deal outside their organization
- **THEN** the system SHALL return HTTP 403 or 404 with a Spanish error
- **AND** SHALL NOT persist the quote

#### Scenario: Totals are server-computed
- **WHEN** a quote is created or its items change
- **THEN** the system SHALL recompute line totals, subtotal, IVA total, and total server-side
- **AND** SHALL NOT accept client-supplied totals

### Requirement: Guarded quote state machine

The system SHALL enforce the quote state machine `borrador → enviada → aprobada | rechazada`, plus `vencida` (validity expiry) and a revise operation creating the next version. Transitions SHALL be guarded: unknown transitions SHALL be rejected; only `enviada` quotes SHALL be approvable/rejectable; `aprobada` SHALL be terminal (new offers require a new version); only `borrador`/`enviada` SHALL be editable. Expiry SHALL be applied by a periodic job based on `valid_until`.

#### Scenario: Quote sent
- **WHEN** a member transitions a `borrador` quote to `enviada`
- **THEN** the quote status SHALL become `enviada`
- **AND** an activity SHALL be recorded on the deal

#### Scenario: Quote approved
- **WHEN** a member transitions an `enviada` quote to `aprobada`
- **THEN** the quote status SHALL become `aprobada`
- **AND** the linked deal `monto` SHALL sync to the quote total
- **AND** an activity SHALL be recorded on the deal

#### Scenario: Quote rejected and revised
- **WHEN** a member rejects an `enviada` quote and requests a revision
- **THEN** the quote status SHALL become `rechazada`
- **AND** a new quote SHALL be created as version+1 with status `borrador` copying the previous items

#### Scenario: Unknown transition rejected
- **WHEN** a transition not present in the state machine is attempted (e.g., approving a `borrador` quote)
- **THEN** the system SHALL reject the transition with an error
- **AND** SHALL NOT change the stored status

#### Scenario: Expired quote becomes vencida
- **WHEN** a periodic job finds an `enviada` quote whose `valid_until` has passed
- **THEN** the quote status SHALL become `vencida`
- **AND** an activity SHALL be recorded on the deal

### Requirement: Deal facturado guard on approval

The system SHALL, when a deal moves to the `facturado` stage, check for an `aprobada` quote on the deal. If none exists, the system SHALL record an advisory activity indicating the quote is not approved; the transition behavior SHALL follow the organization's configured guard mode (default: advisory block — the transition is allowed with a warning activity; when the org flag enables hard mode, the transition SHALL be rejected).

#### Scenario: Deal invoices with approved quote
- **WHEN** a deal with an `aprobada` quote moves to `facturado`
- **THEN** the existing invoice creation SHALL proceed
- **AND** the invoice amount SHALL derive from the approved quote total (via deal `monto` sync)

#### Scenario: Deal without approved quote (advisory mode)
- **WHEN** a deal without an `aprobada` quote moves to `facturado` and the org uses advisory mode
- **THEN** the transition SHALL proceed
- **AND** an activity SHALL warn that no cotización is approved

#### Scenario: Deal without approved quote (hard mode)
- **WHEN** a deal without an `aprobada` quote attempts to move to `facturado` and the org uses hard mode
- **THEN** the transition SHALL be rejected
- **AND** an activity SHALL explain the quote must be approved first
