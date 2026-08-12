## ADDED Requirements

### Requirement: TOTP MFA enrollment

An authenticated member SHALL be able to enroll an authenticator-app (TOTP) factor from settings. Enrollment SHALL use the Stytch B2B TOTP create and authenticate APIs; on completion the member SHALL be shown one-time recovery codes exactly once. Enrollment SHALL be possible regardless of the organization's MFA policy.

#### Scenario: Member enrolls TOTP

- **WHEN** an authenticated member requests TOTP setup
- **THEN** the system SHALL create a TOTP instance via the Stytch API and display the returned `qr_code` image plus the manual secret
- **AND** enrollment SHALL complete only after the member verifies one TOTP code
- **AND** the member SHALL be shown the Stytch-issued recovery codes once

#### Scenario: Optional enrollment applies after verification

- **WHEN** a member enrolls under an `OPTIONAL` org policy
- **THEN** subsequent logins for that member SHALL require TOTP after primary authentication

#### Scenario: Member already has a TOTP registration

- **WHEN** an authenticated member with an existing TOTP registration (`totp_registration_id` on the member object) requests TOTP setup
- **THEN** the system SHALL NOT create a duplicate TOTP instance
- **AND** SHALL surface the existing registration for management (verify/rotate/remove) instead

### Requirement: MFA challenge at login

When primary authentication returns `member_authenticated: false` with an intermediate session token (organization policy or member enrollment requires MFA), the sign-in flow SHALL present a TOTP challenge step. Session cookies SHALL be set only after a successful TOTP (or recovery-code) exchange.

#### Scenario: Required MFA completes

- **WHEN** a member completes primary auth and the org requires MFA
- **THEN** the flow SHALL prompt for a TOTP code
- **AND** on a valid code the flow SHALL exchange the intermediate session for a full session and set the standard session cookies

#### Scenario: Wrong TOTP code

- **WHEN** an invalid TOTP code is submitted
- **THEN** the flow SHALL show an error and keep the member on the challenge step
- **AND** SHALL NOT set session cookies

#### Scenario: Recovery code path

- **WHEN** a member submits a valid recovery code
- **THEN** the recovery-code exchange SHALL complete the MFA flow directly via the Stytch recover endpoint (which returns the session in the same call)
- **AND** on success the flow SHALL set session cookies from that response

#### Scenario: Recovery-code attempts are rate-limited

- **WHEN** recovery-code attempts for a member or from an IP exceed the configured per-member or per-IP limit
- **THEN** the system SHALL reject the additional attempts
- **AND** SHALL record a bounded `mfa_challenge_failed` audit detail (best-effort, non-blocking)

### Requirement: Organization MFA policy management

An admin with `org:manage` SHALL be able to set the organization's MFA policy (`OPTIONAL` default or `REQUIRED_FOR_ALL`) and allowed methods (restricted to `totp` in this change) via a settings UI. The policy SHALL be persisted to Stytch via the organization update API through the Go backend, and outbound calls SHALL be protected by the circuit breaker (503 on breaker-open/unreachable, structured error).

#### Scenario: Admin sets REQUIRED_FOR_ALL

- **WHEN** an admin with `org:manage` sets the policy to `REQUIRED_FOR_ALL` with `allowed_mfa_methods: [totp]`
- **THEN** the Go backend SHALL update the Stytch organization
- **AND** all members SHALL complete TOTP at next login

#### Scenario: Stytch unreachable during policy update

- **WHEN** the policy-update call fails or the circuit breaker is open
- **THEN** the endpoint SHALL return 503 with a structured error
- **AND** the organization's policy SHALL remain unchanged

### Requirement: MFA state lives only in Stytch

Local storage MUST NOT contain TOTP secrets, recovery codes, or MFA material. Local audit rows SHALL record only the bounded event types `mfa_challenge_passed` and `mfa_challenge_failed`.

#### Scenario: No local MFA material

- **WHEN** any MFA operation completes
- **THEN** no secret, recovery code, or session token SHALL be written to the local database or audit stream

#### Scenario: Policy mirror is display-only

- **WHEN** the organization's MFA policy is displayed in settings
- **THEN** the locally read/mirrored policy values SHALL be used for display only
- **AND** SHALL NOT gate any authorization decision — Stytch remains the sole enforcement point for MFA (enforced at session mint)
