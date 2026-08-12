## ADDED Requirements

### Requirement: Login page renders the pre-built Stytch B2B component

The `/auth` login page SHALL render the pre-built Stytch B2B component (`<StytchB2B />` from `@stytch/nextjs/b2b`) in Discovery mode instead of the custom email-only form. The component SHALL render the project-enabled primary methods (email magic link, OAuth social, email OTP, passkeys) and the org discovery surface, and SHALL be configured with `locale: "es"` plus any required strings overrides (matching the product's Colombian-Spanish copy). The custom email form with membership pre-validation (`check-email` + `sendMagicLink` pre-check) SHALL NOT be rendered on `/auth`.

#### Scenario: User visits /auth

- **WHEN** a browser navigates to `/auth`
- **THEN** the page SHALL render `<StytchB2B />` in Discovery mode
- **AND** the page SHALL NOT contain the custom email-only form components

#### Scenario: Member authenticates with magic link

- **WHEN** an existing member submits an email in the Stytch component
- **THEN** Stytch SHALL send the magic link via the discovery send flow
- **AND** the member completes authentication at `/authenticate` as today

#### Scenario: Unknown email with JIT disabled

- **WHEN** a browser submits an email that matches no org with JIT enabled and no membership
- **THEN** the discovery flow SHALL surface no joinable organization for that address
- **AND** the discovery result SHALL NOT reveal any organization to the unknown address (org-existence concealment; the send itself is governed by Stytch's per-email rate limiting — the platform's in-process limiter does not apply to client-side discovery sends)

### Requirement: Discovery org creation stays disabled

The platform's governed `POST /auth/signup` org bootstrap SHALL remain the sole org-creation path. Stytch discovery self-serve org creation SHALL remain disabled in the Stytch dashboard, and the Stytch B2B component SHALL be configured so the "create organization" surface is hidden.

#### Scenario: Discovery shows no create-organization action

- **WHEN** a browser completes the discovery flow with no matching org
- **THEN** the UI SHALL NOT offer self-serve organization creation
- **AND** the user SHALL be directed to the platform signup path or an invite

#### Scenario: Platform signup still bootstraps orgs

- **WHEN** a valid `POST /auth/signup` request is received
- **THEN** the existing bootstrap SHALL create the org + owner member with `SendInvite: true` and seed the trial as before

### Requirement: Per-org JIT domain join (opt-in, domain-restricted)

Organization admins SHALL be able to enable domain-restricted JIT join for their org via the auth-policy settings surface: `email_jit_provisioning: RESTRICTED` together with `email_allowed_domains`, persisted to Stytch via `PUT /v1/b2b/organizations/{organization_id}`. The default SHALL remain `NOT_ALLOWED` for all orgs, including existing ones. JIT-joined members SHALL receive the default `stytch_member` role. JIT org creation SHALL NOT be offered: `email_jit_provisioning` SHALL only ever be set to `RESTRICTED` or `NOT_ALLOWED` (the Stytch org contract exposes no org-creating `ALLOWED` value; discovery org creation stays disabled). Because the setting governs provisioning via Email Magic Link **or** OAuth, enabling domain JIT also enables OAuth JIT for verified provider emails on the allowed domains; the settings UI copy SHALL state this.

#### Scenario: Admin enables domain join

- **WHEN** an admin with `org:manage` enables "allow teammates with @<domain> to join" and saves
- **THEN** the org SHALL be updated with `email_jit_provisioning: RESTRICTED` and the configured `email_allowed_domains`
- **AND** the settings UI SHALL reflect the saved policy (display-only mirror)

#### Scenario: Default remains disabled

- **WHEN** an org has never opted in
- **THEN** its `email_jit_provisioning` SHALL remain `NOT_ALLOWED`
- **AND** discovery SHALL NOT surface the org for non-member emails

#### Scenario: JIT-joined member gets default role

- **WHEN** a member with a matching allowed-domain email authenticates and is JIT-provisioned
- **THEN** the member SHALL hold the default `stytch_member` role
- **AND** role escalation SHALL remain admin-managed

#### Scenario: Auth-policy write is breaker-guarded

- **WHEN** the Stytch circuit breaker is open during a JIT policy update
- **THEN** the update SHALL be rejected with a 503 structured error (`auth_policy_update_unavailable`)
- **AND** the local display mirror SHALL NOT be modified

### Requirement: OAuth social login via Stytch component

Google and Microsoft OAuth providers SHALL be enabled for the project (dashboard-configured) and rendered as login options by the Stytch B2B component. Authentication SHALL complete through the existing `/authenticate` token consumption; no custom OAuth callback endpoints SHALL be introduced.

#### Scenario: Member signs in with Google

- **WHEN** a member selects "Continue with Google" on `/auth` and completes the provider flow
- **THEN** the session SHALL be established with the same session cookies as magic-link login
- **AND** the existing `/authenticate` flow SHALL complete the exchange

### Requirement: Email OTP as an additional primary method

Email OTP SHALL be available as an additional primary authentication method, enabled per organization via the auth-policy surface writing `auth_methods: RESTRICTED` together with `allowed_auth_methods` (SDK values: `magic_link`, `email_otp`, `sso`, `google_oauth`, `microsoft_oauth`) on `PUT /v1/b2b/organizations/{organization_id}`, additive to the existing `magic_link` method. The Stytch B2B component SHALL render the email-OTP method when the org allows it.

#### Scenario: Org adds email OTP

- **WHEN** an admin adds email OTP to the org's allowed primary methods and saves
- **THEN** the org SHALL be written with `auth_methods: RESTRICTED` and `allowed_auth_methods` including `email_otp`
- **AND** the Stytch component SHALL offer "email a one-time code" as a login option
- **AND** existing magic-link login SHALL remain available

#### Scenario: Org without email OTP

- **WHEN** an org's `allowed_auth_methods` does not include `email_otp`
- **THEN** the Stytch component SHALL NOT offer the email-OTP option

#### Scenario: First write preserves existing methods

- **WHEN** an org whose `auth_methods` is currently `ALL_ALLOWED` performs its first policy write
- **THEN** the persisted `allowed_auth_methods` SHALL include the org's current effective method set (at minimum `magic_link`) in addition to the requested changes

### Requirement: /authenticate continuation unchanged

The `/authenticate` page SHALL continue to consume magic-link tokens and drive the MFA (TOTP) and passkey continuation flows exactly as defined by `mfa-totp-enrollment` and `passkeys-sign-in`. This change SHALL NOT alter token consumption, MFA challenge, or passkey registration behavior.

#### Scenario: Magic link token still consumed at /authenticate

- **WHEN** a member opens the magic link
- **THEN** `/authenticate` SHALL consume the token and establish the session as before
- **AND** MFA-required orgs SHALL still route through the TOTP challenge step
