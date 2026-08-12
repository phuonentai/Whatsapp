# mercadopago-webhooks Specification

## Purpose
TBD - created by archiving change fix-mercadopago-provider-seams. Update Purpose after archive.
## Requirements
### Requirement: MercadoPago subscription events persist mapped statuses
The system SHALL persist domain-mapped statuses from MP subscription webhooks and refresh syncs: `authorized` → `active`, `paused` → `past_due`, `cancelled` → `canceled`.

#### Scenario: Refresh sync maps raw status
- **WHEN** a refresh or sync reads an MP preapproval with status `authorized` or `paused`
- **THEN** the local subscription SHALL be stored with the mapped status (`active` / `past_due`)

### Requirement: MercadoPago payment events dispatch the payment id
The system SHALL dispatch `payment_created`, `payment_updated`, and `payment_approved` events to the client-payments handler with the payment id from `data.id`, not the notification id.

#### Scenario: Payment event carries data id
- **WHEN** an MP payment event arrives with `data.id`
- **THEN** the client-payments handler SHALL receive that payment id

### Requirement: Webhook ingress is reachable at the advertised path
The system SHALL serve the per-provider webhook endpoints at `/api/v1/webhooks/polar` and `/api/v1/webhooks/mercadopago` (single `/api` prefix), matching the whatsapp webhook pattern.

#### Scenario: Polar webhook URL resolves
- **WHEN** a client POSTs to `/api/v1/webhooks/polar`
- **THEN** the request SHALL reach the Polar webhook handler
- **AND** NOT 404 due to a duplicated `/api` prefix

#### Scenario: MP webhook URL resolves
- **WHEN** a client POSTs to `/api/v1/webhooks/mercadopago`
- **THEN** the request SHALL reach the MP webhook handler
- **AND** NOT 404 due to a duplicated `/api` prefix

### Requirement: Webhook signature verification rejects replays
The system SHALL verify the MercadoPago `x-signature` header in the `ts=,v1=` format and SHALL reject signatures whose timestamp falls outside a freshness window (e.g. 5 minutes) or whose header is not in that format.

#### Scenario: Stale signature rejected
- **WHEN** a webhook payload carries an `x-signature` whose `ts` is older than the freshness window
- **THEN** the request SHALL be rejected with 401

#### Scenario: Non-conforming header rejected
- **WHEN** the `x-signature` header is not in `ts=,v1=` format
- **THEN** the request SHALL be rejected with 401

