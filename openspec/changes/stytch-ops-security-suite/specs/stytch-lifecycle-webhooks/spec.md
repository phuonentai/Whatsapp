## ADDED Requirements

### Requirement: Stytch webhook ingress with signature verification

The system SHALL expose `POST /api/webhooks/stytch` accepting Stytch (Svix standard) webhooks. Every request SHALL be verified using the Svix signing scheme before any processing: HMAC-SHA256 over `{Webhook-Id}.{Webhook-Timestamp}.{payload}` with the `whsec_` webhook secret, constant-time comparison, and a bounded replay window on `Webhook-Timestamp`. Unverified requests SHALL receive a 401 and SHALL NOT trigger any DB mutation, audit write, or outbound call.

#### Scenario: Verified webhook processed

- **WHEN** a request arrives with a valid `Webhook-Id`, `Webhook-Timestamp`, and `Webhook-Signature` for the current secret
- **THEN** the system SHALL process the event

#### Scenario: Invalid signature rejected

- **WHEN** a request arrives with a malformed or wrong signature
- **THEN** the system SHALL respond 401
- **AND** SHALL NOT mutate the database or call any API

#### Scenario: Stale timestamp rejected

- **WHEN** a request's `Webhook-Timestamp` falls outside the replay window
- **THEN** the system SHALL reject the request

### Requirement: Idempotent webhook processing

The system SHALL process each webhook `event_id` at most once. Deduplication SHALL be enforced transaction-isolated: a dedup row is written in the same transaction as the event's resulting writes, so replays and parallel deliveries SHALL NOT create duplicate audit rows or duplicate effects.

#### Scenario: Replayed event does not duplicate

- **WHEN** Stytch re-delivers an already-processed `event_id`
- **THEN** the system SHALL acknowledge the delivery
- **AND** SHALL NOT create a second audit row or duplicate effect

#### Scenario: Concurrent deliveries race safely

- **WHEN** two deliveries of the same `event_id` arrive concurrently
- **THEN** exactly one SHALL produce the writes
- **AND** the other SHALL acknowledge without effect

### Requirement: Lifecycle events feed the governance audit stream

Verified member/org lifecycle events SHALL append governance audit rows (`tipo=sistema`) to the organization audit stream. The system SHALL treat the webhook payload as a trigger and re-fetch authoritative state via the Stytch API (breaker-guarded) before writing, never trusting the payload as authoritative state. Recognized event families: `direct.member.create/update/delete`, `direct.organization.create/update`, `dashboard.member.*`, `scim.member.*`. Mapped event types SHALL include `member_invited`, `member_role_changed`, `member_removed`, `organization_updated`, `member_deprovisioned`, with bounded detail (member id, organization id, role before/after where applicable, timestamp) and no credential or payload material.

#### Scenario: Member invited records governance event

- **WHEN** a verified `direct.member.create` event (invite) arrives
- **THEN** the system SHALL append a `member_invited` audit row for the organization

#### Scenario: Role change records before/after

- **WHEN** a verified `direct.member.update` event reflects a role change
- **THEN** the system SHALL append a `member_role_changed` row with the role before/after

#### Scenario: SCIM deprovisioning records member removal

- **WHEN** a verified `scim.member.*` event reflects deprovisioning
- **THEN** the system SHALL append a `member_deprovisioned` audit row

#### Scenario: Unknown event acked and logged

- **WHEN** an event family or type is not recognized
- **THEN** the system SHALL log it
- **AND** SHALL acknowledge the delivery without error

#### Scenario: Fetch failure on breaker open

- **WHEN** the Stytch circuit breaker is open during the authoritative re-fetch
- **THEN** the system SHALL log the failure and acknowledge the event
- **AND** SHALL NOT write a governance row based on the unverified payload

### Requirement: Webhook secret handling

The `whsec_` webhook secret SHALL be configured via environment/secrets management, SHALL NOT be stored in the database, SHALL NOT appear in logs, and SHALL be rotatable without code changes.

#### Scenario: Secret rotation

- **WHEN** the webhook secret is rotated in the Stytch dashboard and the environment
- **THEN** verification SHALL use the new secret
- **AND** no code change SHALL be required
