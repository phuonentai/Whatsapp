## ADDED Requirements

### Requirement: Mock invoicing provider server in the e2e environment

The e2e environment SHALL boot a mock invoicing provider server (Go command `cmd/mock-siigo`) that implements the Siigo adapter surface: OAuth token grant at the configured token URL, no company endpoint (404), paginated customer list, invoice creation with consecutive numbering and Idempotency-Key deduplication (same key returns the previously created invoice), and invoice status lookup. The backend SHALL be configured with `SIIGO_BASE_URL`, `SIIGO_TOKEN_URL`, and `SIIGO_WEBHOOK_SECRET` pointing at the mock; no e2e test SHALL make a network call to the real Siigo API.

#### Scenario: E2E backend talks only to the mock provider

- **WHEN** the e2e stack boots and a test drives connect, import, or invoice creation
- **THEN** all provider traffic SHALL hit the mock server
- **AND** the real Siigo API SHALL NOT be reachable from the e2e configuration

#### Scenario: Idempotency-Key honored by the mock

- **WHEN** two invoice POSTs carry the same Idempotency-Key for the same organization and deal
- **THEN** the mock SHALL return the first created invoice and SHALL NOT create a duplicate

### Requirement: Siigo test organization in the seed

The seed command SHALL create a dedicated `test-org-siigo` organization (Pro plan) with an admin account and a member account, reserved for the Siigo onboarding e2e suite, so scenario state (connection rows, imports) does not leak into the general-purpose seeded organizations.

#### Scenario: Seed provides the Siigo org and RBAC accounts

- **WHEN** `cmd/seed-e2e` runs
- **THEN** `test-org-siigo` SHALL exist with admin and member accounts
- **AND** the general-purpose seeded orgs SHALL remain available for their existing suites
