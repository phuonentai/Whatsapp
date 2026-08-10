## ADDED Requirements

### Requirement: Numeration continuity confirmation

The system SHALL read the organization's active DIAN numeration (resolución, prefijo, current next number) from the connected Siigo provider and expose it for confirmation. Confirming numeration SHALL store a snapshot (`invoicing.org_numerations`) and SHALL advance the connection state from `connected` to `numeracion_ok`. The snapshot SHALL record the confirmed next number and timestamp. Invoice creation SHALL respect the confirmed numeration such that issued numbers remain consecutive with the client's existing Siigo sequence.

#### Scenario: Organization confirms numeration and advances

- **WHEN** an organization with state `connected` reads and confirms the numeration returned by Siigo
- **THEN** the system SHALL store the numeration snapshot (resolución, prefijo, next number)
- **AND** SHALL advance the connection to `numeracion_ok`
- **AND** SHALL NOT create invoices until numeration is confirmed

#### Scenario: Numeration read fails does not advance

- **WHEN** the numeration read from Siigo fails
- **THEN** the system SHALL return an error
- **AND** SHALL NOT change the connection state or store a snapshot

### Requirement: Invoice numbering continuity modes

The system SHALL create provider invoices under the organization's confirmed numeration. When the provider auto-assigns consecutive numbers, the system SHALL pass no number (mode `auto`) and rely on the provider's sequence. When the provider requires a number, the system SHALL fetch the next available number for the organization (single-flight per org) and include it in the invoice payload (mode `manual`), retrying once if the provider reports a number conflict.

#### Scenario: Auto mode relies on provider sequence

- **WHEN** invoice creation runs in mode `auto`
- **THEN** the system SHALL NOT supply an invoice number
- **AND** the provider's assigned number SHALL be stored with the invoice

#### Scenario: Manual mode fetches next number and retries on conflict

- **WHEN** invoice creation runs in mode `manual`
- **THEN** the system SHALL obtain the next number for the organization exactly once per creation attempt (single-flight)
- **AND** SHALL include it in the invoice payload
- **AND** SHALL retry once with a freshly fetched number if the provider rejects the first as a conflict

### Requirement: Customer import from provider with NIT deduplication

The system SHALL import provider customers into the CRM as companies with linked contacts, deduplicating by normalized NIT within the organization (`(organization_id, nit)`). Import SHALL expose a preview (counts of new, existing, duplicate, and without-NIT records) before any write, and a confirm operation that commits the batch. Imported companies SHALL respect the existing `(organization_id, name)` uniqueness rule.

#### Scenario: Import preview reports counts without writing

- **WHEN** an organization requests an import preview
- **THEN** the system SHALL pull provider customers and report counts: nuevos, existentes, duplicados por NIT, sin NIT
- **AND** SHALL NOT write any company or contact row

#### Scenario: Confirmed import upserts companies and contacts

- **WHEN** an organization confirms an import batch
- **THEN** the system SHALL upsert companies and linked contacts keyed by normalized NIT
- **AND** SHALL record the import run (counts, timestamp) in `invoicing.import_runs`
- **AND** repeated imports SHALL NOT duplicate records (idempotent by NIT)

### Requirement: Delta customer sync

The system SHALL re-pull provider customers on demand and on a scheduled interval (nightly), applying idempotent upserts by NIT without preview. Delta sync SHALL record an import run for every execution.

#### Scenario: On-demand delta sync updates existing customers

- **WHEN** an organization triggers a delta sync
- **THEN** the system SHALL pull provider customers and upsert changes by NIT
- **AND** SHALL record the run in `invoicing.import_runs`
- **AND** SHALL NOT require confirmation

### Requirement: Sandbox test invoice before go-live

The system SHALL provide a test-invoice operation that creates a sandbox invoice for the organization, awaits its status through the existing status-sync path (webhook or polling fallback), and SHALL advance the connection to `sandbox_ok` only when the invoice reaches a valid state. The operation SHALL be rejected with HTTP 400 when the provider is not in sandbox mode.

#### Scenario: Successful sandbox invoice advances onboarding

- **WHEN** the provider is sandboxed and a test invoice reaches a valid status
- **THEN** the system SHALL store the test invoice (no deal required)
- **AND** SHALL advance the connection from `numeracion_ok` to `sandbox_ok`

#### Scenario: Test invoice rejected outside sandbox

- **WHEN** the provider is configured for production and a test-invoice operation is requested
- **THEN** the system SHALL return HTTP 400
- **AND** SHALL NOT create any invoice
