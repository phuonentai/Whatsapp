# Passkeys Sign-In

## Why

The sign-in path is email-magic-link only. Passkeys (WebAuthn/FIDO2) are the 2026 baseline for phishing-resistant authentication: Microsoft is defaulting passkeys in Entra ID in Sep 2026, FIDO Alliance research shows 87% of large orgs deploying them, and "do you support passkeys?" is now an enterprise procurement gate. Stytch B2B fully supports passkeys (`/v1/b2b/webauthn/*` + member WebAuthn management). The product's invite-only B2B positioning (multi-org members, admins, approvers) makes passkeys a high-value conversion + security win, while magic links remain the fallback for new/unenrolled devices.

## What Changes

- **Registration:** settings → profile → Security shows "Add a passkey"; uses Stytch B2B WebAuthn register APIs (`register/start` → browser credential creation → `register`), bound to the app origin (RP ID = app domain, chosen once — see Non-Goals).
- **Sign-in:** on `/auth`, after email resolution (existing membership validation), members with passkeys can authenticate with a passkey via Stytch B2B WebAuthn authenticate APIs (`authenticate/start` → browser assertion → `authenticate`); on success the existing session cookies are set. Magic link remains the fallback and the primary path for unenrolled devices.
- **Management:** member can list and delete their own passkeys (Stytch member WebAuthn endpoints), scoped server-side to the authenticated session.
- **SSOT:** passkey credentials, attestation, and state live exclusively in Stytch; the local DB stores nothing passkey-related.
- **Resilience:** all passkey Stytch calls run through a new frontend circuit-breaker wrapper (threshold 5 / timeout 10s / half-open probe 2, mirroring the Go adapter contract); passkey challenge-start is rate-limited per email and per IP; the browser WebAuthn ceremony is bounded by an abort timeout with a defined outcome taxonomy (user-cancel vs Stytch failure vs network failure).

## Revision (council, 2026-08-12)

This revision addresses the council verdict on the original design (`VERDICT.md`, REJECTED). Required changes folded in:

1. **Breaker contradiction resolved:** the original design claimed passkey calls flow through "the existing auth adapter circuit breaker" — none exists in the Next.js Stytch path (`getStytchB2BClient()` in `lib/auth/stytch/server.ts` constructs `new Stytch.B2BClient()` directly; the two-tier breaker lives only in the Go adapter). The design now introduces a frontend breaker wrapper as an explicit new seam and specifies its failure contract (breaker-open → structured 503, no session issuance, magic-link fallback).
2. **Rate limiting added:** `startPasskeyAuthentication` / `createPasskeyRegistration` reuse the in-process sliding-window limiter pattern (per-email + per-IP, env-configurable), closing the challenge-start spam vector.
3. **Server-side ownership binding:** list/delete derive `member_id` / `organization_id` from the verified session; `startPasskeyAuthentication` re-resolves the member server-side from the email (never accepts opaque client-supplied IDs).
4. **MFA/primary composition completed:** `member_authenticated: false` responses propagate `intermediate_session_token` + `mfa_required` (+ `primary_required`) to the MFA challenge step (per `mfa-totp-enrollment`), mirroring `consumeMagicLink`; no cookies are set.
5. **Ceremony + observability:** abort-timeout on `navigator.credentials.create/get`; distinct handling for user-cancel (silent magic-link fallback) vs Stytch failure vs network failure; passkey audit writes are best-effort and non-blocking.

## Capabilities

### New Capabilities
- `passkeys-sign-in`: passkey registration, passkey sign-in, and self-service passkey management.

### Modified Capabilities
- None.

## Impact

- **Frontend:** new passkey actions (`lib/actions/auth/passkeys.ts`), `/auth` passkey branch, settings profile passkey section (register/list/delete), conditional-UI handling where supported, frontend breaker wrapper (`lib/auth/stytch/breaker.ts`), passkey rate limiter (`lib/auth/passkey-limiter.ts`).
- **Backend:** none required — WebAuthn is orchestrated by the Stytch SDK from the Next.js server (existing pattern: server actions call Stytch directly). The breaker wrapper is a NEW frontend seam; the existing Go two-tier breaker is untouched. RP ID / domain config is Stytch-dashboard side.
- **Dependencies:** none new (`@stytch/nextjs` + browser WebAuthn APIs).
- **Stytch:** passkeys product must be enabled in the Stytch project; RP ID configured in the dashboard; per-member WebAuthn instances become tenant state.

## Rollback

- **Git:** revert frontend actions/UI; the email magic-link path is untouched and remains the default.
- **Stytch tenant policy state:** disable the passkeys product or delete member WebAuthn instances via `DELETE /v1/b2b/member_webauthn/{member_webauthn_id}`; no org-level policy change required (passkeys are additive to the existing login). NOTE (assumption, verify in the Stytch test project during E2E): whether disabling the product revokes already-enrolled credentials is undocumented — plan for per-member cleanup if immediate revocation is required.

## Non-Goals

- NOT changing the RP ID after launch (invalidates all passkeys) — the app domain is chosen deliberately before rollout.
- NOT implementing SSO-federated passkey ownership (tenant IdP ceremony) — out of scope; relevant only when SSO connections are added.
- NOT storing passkeys, attestation objects, or session material locally (SSOT constitution).
- NOT replacing magic links — passkeys are an additional factor/primary path, magic links remain the fallback.
- NOT retrofitting the breaker wrapper onto existing magic-link actions (`sendMagicLink`/`consumeMagicLink` still call the SDK directly) — out of scope for this change; the wrapper becomes the seam for future auth actions.
