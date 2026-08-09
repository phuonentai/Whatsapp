## ADDED Requirements

### Requirement: Invoice creation triggered by a deal reaching the invoicing stage

The system SHALL create an electronic invoice for a deal when the deal's stage changes to `facturado`. Creation SHALL be idempotent: at most one invoice per deal, enforced by a unique constraint on `(organization_id, deal_id)`. A re-trigger (duplicate stage-change event) SHALL return the existing invoice and make no change. Invoice data (amount, customer NIT/name) SHALL be derived from the deal, its linked company/contact, and the organization's billing configuration.

#### Scenario: Deal moves to facturado creates one invoice

- **WHEN** a deal's stage changes to `facturado`
- **THEN** the system SHALL call the resolved invoicing provider to create an invoice
- **AND** SHALL store the invoice result in `invoicing.invoices` with status `pending`
- **AND** SHALL record an activity on the deal

#### Scenario: Repeated stage-change events do not duplicate invoices

- **WHEN** a second `DealStageChanged` event for the same deal arrives after an invoice already exists
- **THEN** the system SHALL return the existing invoice
- **AND** SHALL NOT create a new invoice at the provider

#### Scenario: Other stage changes do not create invoices

- **WHEN** a deal's stage changes to a stage other than `facturado`
- **THEN** the system SHALL NOT create an invoice

### Requirement: Per-organization invoicing provider routing

The system SHALL resolve the invoicing provider per organization through an `InvoiceRouter`. The default and only implemented provider SHALL be Siigo; the router SHALL expose a seam such that a second provider (e.g., Alegra) can be added without changing the invoicing service. Domain models SHALL NOT import provider SDKs or transport packages; provider adapters SHALL implement the domain `InvoicingProvider` interface.

#### Scenario: Organization routes to Siigo

- **WHEN** the invoicing service creates an invoice for an organization with no explicit provider preference
- **THEN** the router SHALL delegate to the Siigo adapter
- **AND** the invoice SHALL be stored scoped by `organization_id`

#### Scenario: Unsupported provider fails closed

- **WHEN** an organization's configured invoicing provider is not recognized
- **THEN** invoice creation SHALL fail with an error and SHALL NOT write a partial invoice record

### Requirement: Invoice status sync via signed webhook

The system SHALL expose `POST /api/v1/webhooks/siigo` that accepts Siigo invoice-status notifications. The handler SHALL verify the webhook signature before any database mutation; a missing or invalid signature SHALL return HTTP 401 with error code `invalid_signature` and no state change. Processing SHALL be idempotent and transaction-isolated: the status update SHALL be applied only if the event's invoice state is newer than the stored state.

#### Scenario: Valid webhook updates invoice status

- **WHEN** a signed notification arrives with a valid invoice status (e.g., `valid`, CUFE, PDF URL)
- **THEN** the system SHALL update the matching `invoicing.invoices` row
- **AND** return HTTP 200

#### Scenario: Invalid signature is rejected

- **WHEN** a request arrives without a valid signature
- **THEN** the system SHALL return HTTP 401 with error code `invalid_signature`
- **AND** SHALL NOT mutate any invoice row

#### Scenario: Status regressions are ignored

- **WHEN** a notification carries a status older than the stored state
- **THEN** the system SHALL keep the newer stored state
- **AND** SHALL NOT overwrite it

### Requirement: Polling fallback for invoice status

The system SHALL poll the provider for the status of invoices that have not reached a final state, as a safety net for missed webhook notifications. Polling SHALL NOT trigger the WhatsApp notification more than once per status transition.

#### Scenario: Stuck invoice reconciled by polling

- **WHEN** an invoice remains in a non-final status and the provider reports a final status during polling
- **THEN** the system SHALL update the stored status to the provider-reported status
- **AND** SHALL notify the contact at most once for that transition

### Requirement: Provider token cache without credential persistence

The Siigo adapter SHALL obtain access tokens via OAuth2 `client_credentials` and cache them in memory with a TTL not exceeding 300 seconds. Client secrets and tokens SHALL be loaded from environment configuration only and SHALL NOT be stored in PostgreSQL, logs, or any persistent local store. Expired tokens SHALL be refreshed transparently with a single-flight refresh and one retry on HTTP 401.

#### Scenario: Token refreshed on expiry

- **WHEN** the cached access token is expired at the time of an invoice call
- **THEN** the adapter SHALL obtain a new token using the stored client credentials
- **AND** SHALL complete the invoice call

#### Scenario: No credentials persisted

- **WHEN** the adapter stores access tokens
- **THEN** the token SHALL reside only in the in-memory cache
- **AND** SHALL NOT be written to the database or logs

### Requirement: WhatsApp invoice notification with payment link

The system SHALL notify the deal's contact inside WhatsApp when an invoice is created (invoice link) and when it becomes valid (CUFE/PDF link), using an approved transactional template and reusing the existing WhatsApp send path. The notification SHALL include the MercadoPago payment link when available. Invoice notifications are transactional (required to fulfill the sale) and SHALL be sent even to contacts with `consent_status = 'withdrawn'`, but SHALL NOT contain promotional content.

#### Scenario: Invoice created triggers WhatsApp message

- **WHEN** an invoice is created for a deal linked to a WhatsApp contact
- **THEN** the system SHALL send the invoice link via the `factura_lista` template
- **AND** SHALL include the payment link when available

#### Scenario: Transactional message allowed for withdrawn contacts

- **WHEN** a deal's contact has `consent_status = 'withdrawn'` and an invoice becomes due
- **THEN** the system SHALL send the transactional invoice notification
- **AND** SHALL NOT include promotional content

#### Scenario: Template failure does not fail invoicing

- **WHEN** the WhatsApp template send fails
- **THEN** the invoice creation/status update SHALL remain committed
- **AND** the failure SHALL be logged as a warning
