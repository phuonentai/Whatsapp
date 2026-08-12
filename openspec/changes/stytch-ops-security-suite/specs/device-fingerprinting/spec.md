## ADDED Requirements

### Requirement: Client captures a Stytch device fingerprint

The login surface SHALL call the Stytch B2B fingerprint method (`stytch.b2b.fingerprint()` from the pinned `@stytch/vanilla-js`) when a session starts, and SHALL submit the resulting `device_id` to `POST /api/risk/device-fingerprint` (authenticated, same-origin). The backend SHALL persist the `device_id` per member-organization.

#### Scenario: Login captures a fingerprint

- **WHEN** a member begins a login session
- **THEN** the client SHALL obtain a `device_id` via `stytch.b2b.fingerprint()`
- **AND** SHALL submit it to the authenticated fingerprint endpoint

#### Scenario: Fingerprint persists per member-org

- **WHEN** the backend receives a fingerprint submission
- **THEN** the `device_id` SHALL be stored associated with the authenticated member and organization

#### Scenario: Unauthenticated submission rejected

- **WHEN** an unauthenticated request hits the fingerprint endpoint
- **THEN** the system SHALL return 401

### Requirement: Fingerprints contain no PII and respect the free budget

Fingerprint rows SHALL contain only the opaque Stytch `device_id`, member id, organization id, and timestamp — no PII and no other device-identifying material. The system SHALL track the fingerprint volume against the 10,000/month free budget and SHALL log a warning when approaching the limit.

#### Scenario: Fingerprint row is PII-free

- **WHEN** a fingerprint row is written
- **THEN** it SHALL contain only `device_id`, member id, organization id, and timestamp

#### Scenario: Budget warning near the limit

- **WHEN** fingerprint volume approaches the 10,000/month budget
- **THEN** the system SHALL log a warning

### Requirement: Fingerprints surface for support triage

The stored fingerprint SHALL be shown on the member profile for support triage. Fingerprint data SHALL be exposed to the risk/rate-limiting surfaces as an optional per-device dimension; existing limiter behavior SHALL NOT change in this capability.

#### Scenario: Member profile shows fingerprint

- **WHEN** a support member with `org:manage` views a member profile
- **THEN** the profile SHALL display the member's stored fingerprint(s)
