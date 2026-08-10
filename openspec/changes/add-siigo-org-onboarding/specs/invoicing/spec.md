## ADDED Requirements

### Requirement: Per-organization invoicing connection with onboarding state

The system SHALL store a single invoicing connection per organization in `invoicing.org_connections` with: provider (`siigo` | `none`), onboarding status, Siigo company data (NIT, company name), encrypted API credentials, last error, and timestamps. The onboarding status SHALL follow the state machine `none → awaiting_setup | connected → numeracion_ok → sandbox_ok → live`, with terminal state `invoicing_disabled` and reversible state `paused`. Every state transition SHALL be guarded: unknown transitions SHALL be rejected.

#### Scenario: Organization connects Siigo and advances through onboarding

- **WHEN** an organization's connection is in `none` state and the owner supplies valid Siigo credentials
- **THEN** the system SHALL validate the credentials against Siigo
- **AND** SHALL verify the Siigo company NIT matches the organization's NIT
- **AND** SHALL store the connection with status `connected`
- **AND** SHALL store the credentials encrypted at rest

#### Scenario: NIT mismatch rejects the connection

- **WHEN** the Siigo company NIT does not match the organization's NIT during connect
- **THEN** the system SHALL reject the connection with an explicit NIT-mismatch error
- **AND** SHALL NOT persist credentials or advance the state

#### Scenario: Invalid Siigo credentials are rejected

- **WHEN** credentials fail Siigo validation during connect
- **THEN** the system SHALL return an authentication error
- **AND** SHALL NOT persist credentials or advance the state

#### Scenario: State transition guard rejects unknown transitions

- **WHEN** a transition not present in the state machine is attempted (e.g., `live` directly from `none`)
- **THEN** the system SHALL reject the transition with an error
- **AND** SHALL NOT change the stored state

### Requirement: Provider routing with explicit none state

The system SHALL resolve the invoicing provider per organization through the `InvoiceRouter`. An organization with a connection provider of `none` SHALL route to an explicit no-op provider (no provider call, no error); an organization with an unrecognized provider SHALL fail closed.

#### Scenario: Organization without a provider routes to no-op

- **WHEN** the invoicing service resolves the provider for an organization whose connection provider is `none`
- **THEN** the router SHALL return the explicit no-op provider
- **AND** the service SHALL make no provider call and return an informative result

#### Scenario: Unknown provider fails closed

- **WHEN** an organization's configured provider value is not `siigo` or `none`
- **THEN** invoice creation SHALL fail with an error and SHALL NOT write a partial invoice record

### Requirement: Deal-stage invoicing gated on connection state

The system SHALL gate deal-stage invoice creation on the organization's connection state. Invoice creation SHALL be attempted only when the state is `live` (or `paused`-resumed from live); any other state SHALL record a deal activity `Facturación no activa` and SHALL NOT call the provider or create an invoice.

#### Scenario: Deal reaches facturado while connection is not live

- **WHEN** a deal's stage changes to `facturado` and the organization's connection state is not `live`
- **THEN** the system SHALL NOT call the invoicing provider
- **AND** SHALL NOT create an invoice
- **AND** SHALL record a deal activity indicating invoicing is inactive

#### Scenario: Deal reaches facturado while connection is live

- **WHEN** a deal's stage changes to `facturado` and the organization's connection state is `live`
- **THEN** the system SHALL create the invoice through the resolved provider as specified by the existing deal-stage requirement

### Requirement: Encrypted per-org credential storage

The system SHALL store Siigo client credentials per organization encrypted at rest using AES-256-GCM with an envelope key loaded from environment configuration (`SIIGO_MASTER_KEY`). Credentials SHALL NOT be stored in plaintext, SHALL NOT appear in logs or error messages, SHALL NOT be returned by any API response, and SHALL be decryptable only in the Siigo adapter's credential resolution path.

#### Scenario: Credentials stored encrypted and never exposed

- **WHEN** an organization connects Siigo and credentials are persisted
- **THEN** the stored row SHALL contain only ciphertext and nonce for the secret values
- **AND** no API endpoint SHALL return the credential values or ciphertext
- **AND** no log line SHALL contain credential material

#### Scenario: Adapter decrypts credentials for outbound calls

- **WHEN** the Siigo adapter needs credentials for an outbound OAuth call
- **THEN** the system SHALL decrypt the stored ciphertext with the envelope key
- **AND** SHALL use the plaintext only in memory for the duration of the call

### Requirement: Assisted setup path for organizations without Siigo

The system SHALL support provisioning a Siigo connection on behalf of an organization through an admin-scoped operation. An organization marked for assisted setup SHALL be in state `awaiting_setup` until an admin provisions credentials, after which it SHALL follow the same transitions as a self-connected organization.

#### Scenario: Admin provisions credentials for an awaiting organization

- **WHEN** an admin with the appropriate role provisions Siigo credentials for an organization in `awaiting_setup`
- **THEN** the system SHALL validate and store the credentials as in the self-serve connect flow
- **AND** SHALL advance the organization to `connected`

#### Scenario: Assisted setup requires admin scope

- **WHEN** a non-admin member attempts the assisted-provisioning operation
- **THEN** the system SHALL reject the operation with HTTP 403

### Requirement: Invoicing kill-switch

The system SHALL provide pause and resume operations on an organization's invoicing connection. Pausing SHALL make deal-stage invoicing behave as non-live (activity recorded, no provider call) while preserving the connection and its state; resuming SHALL restore the previous state.

#### Scenario: Paused organization does not invoice

- **WHEN** an organization's invoicing is paused and a deal reaches `facturado`
- **THEN** the system SHALL NOT call the provider and SHALL record the inactive-invoicing activity

#### Scenario: Resumed organization invoices again

- **WHEN** a paused organization resumes invoicing
- **THEN** the connection SHALL return to its pre-pause state
- **AND** subsequent `facturado` deals SHALL invoice normally
