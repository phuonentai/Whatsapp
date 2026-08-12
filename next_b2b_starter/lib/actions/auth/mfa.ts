"use server";

import { cookies, headers } from "next/headers";

import { getMemberSession, getStytchB2BClient } from "@/lib/auth/stytch/server";
import { mapAuthErrorToDetail, recordAuthAudit } from "@/lib/auth/audit";
import {
  SESSION_COOKIE_NAME,
  SESSION_JWT_COOKIE_NAME,
} from "@/lib/auth/constants";
import {
  getSessionDurationMinutes,
  getCookieConfig,
  getSecureCookieConfig,
} from "@/lib/auth/server-constants";
import { checkMfaRateLimit } from "@/lib/auth/mfa-limiter";
import {
  createActionError,
  createActionSuccess,
  type ActionResult,
} from "@/lib/utils/server-action-helpers";

export interface AuthenticateTotpInput {
  /**
   * Single-use token minted by primary auth when MFA is required. This is the
   * authoritative credential bound to the member by Stytch; `memberId` /
   * `organizationId` are context only (used for the Stytch request path, the
   * rate limiter key, and audit attribution — never trusted for authorization).
   */
  intermediateSessionToken: string;
  /** TOTP code from the authenticator app (TOTP path). */
  code?: string;
  /** One-time recovery code (recovery path; mutually exclusive with `code`). */
  recoveryCode?: string;
  memberId?: string;
  organizationId?: string;
}

export interface CreateTotpResult {
  status: "created" | "existing";
  /** Stytch `totp_registration_id` for the member's (only) TOTP instance. */
  totpRegistrationId: string;
  /** Server-generated QR code image (data URI) — rendered as-is, no QR lib. */
  qrCode?: string;
  /** Manual secret shown alongside the QR code. */
  secret?: string;
  /** One-time recovery codes; shown exactly once after verification. */
  recoveryCodes?: string[];
}

export interface VerifyTotpEnrollmentInput {
  code: string;
}

/** Generic error message for throttled MFA attempts. */
const GENERIC_MFA_ERROR =
  "We couldn't verify that. Please try again later.";

/**
 * Derive the client IP from proxy headers (same trust order as the magic-link
 * action): `x-forwarded-for` first entry -> `x-real-ip` -> localhost fallback.
 */
function deriveClientIp(headerStore: {
  get(name: string): string | null;
}): string {
  const forwardedFor = headerStore.get("x-forwarded-for");
  if (forwardedFor) {
    const firstEntry = forwardedFor.split(",")[0]?.trim();
    if (firstEntry) {
      return firstEntry;
    }
  }
  const realIp = headerStore.get("x-real-ip");
  if (realIp) {
    return realIp;
  }
  return "127.0.0.1";
}

/** Extract a safe, user-displayable message from a Stytch API error. */
function extractErrorMessage(error: unknown): string {
  if (
    error &&
    typeof error === "object" &&
    "error_message" in error &&
    typeof error.error_message === "string" &&
    error.error_message.length > 0
  ) {
    return error.error_message;
  }
  if (error instanceof Error && error.message) {
    return error.message;
  }
  return GENERIC_MFA_ERROR;
}

async function setSessionCookies(
  sessionToken: string | undefined,
  sessionJwt: string | undefined,
  durationMinutes: number
): Promise<void> {
  const cookieStore = await cookies();
  const maxAgeSeconds = durationMinutes * 60;

  if (sessionToken) {
    cookieStore.set(SESSION_COOKIE_NAME, sessionToken, {
      ...getSecureCookieConfig(),
      maxAge: maxAgeSeconds,
    });
  }
  if (sessionJwt) {
    cookieStore.set(SESSION_JWT_COOKIE_NAME, sessionJwt, {
      ...getCookieConfig(),
      maxAge: maxAgeSeconds,
    });
  }
}

/**
 * Create a TOTP enrollment for the signed-in member (duplicate-guarded).
 *
 * Reads the member's current session server-side (the session JWT is an
 * httpOnly cookie — the client can never supply it). Before any Stytch create
 * call, the member's `totp_registration_id` is checked: if a registration
 * already exists, no create call is made and the caller is directed to manage
 * the existing instance instead of creating a duplicate (Stytch
 * multi-registration semantics are undocumented; guarded here).
 */
export async function createTotp(): Promise<ActionResult<CreateTotpResult>> {
  const session = await getMemberSession();
  if (!session) {
    return createActionError(
      "You must be signed in to set up two-factor authentication."
    );
  }

  const memberId = session.member?.member_id;
  const organizationId = session.organization?.organization_id;
  if (!memberId || !organizationId) {
    return createActionError(
      "Your session is missing member or organization context."
    );
  }

  // Duplicate guard: never create a second TOTP instance for a member that
  // already holds one (design D2, verdict #4).
  if (session.member?.totp_registration_id) {
    return createActionSuccess({
      status: "existing",
      totpRegistrationId: session.member.totp_registration_id,
    });
  }

  try {
    const client = getStytchB2BClient();
    const result = await client.totps.create({
      organization_id: organizationId,
      member_id: memberId,
      session_jwt: session.session_jwt,
    });

    return createActionSuccess({
      status: "created",
      totpRegistrationId: result.totp_registration_id,
      qrCode: result.qr_code,
      secret: result.secret,
      recoveryCodes: result.recovery_codes,
    });
  } catch (error) {
    console.error("[MFA] TOTP create failed:", error);
    return createActionError(extractErrorMessage(error));
  }
}

/**
 * Verify a TOTP code against a pending enrollment (settings flow).
 *
 * Confirms the member scanned the QR code / entered the secret. Uses the
 * member's live session (`session_jwt`); on success sets `mfa_enrolled: true`
 * so the member is required to complete MFA on subsequent logins even under
 * an `OPTIONAL` org policy (spec: optional enrollment applies after
 * verification). No cookies are set — the member is already signed in.
 */
export async function verifyTotpEnrollment(
  input: VerifyTotpEnrollmentInput
): Promise<ActionResult<{ totpRegistrationId: string }>> {
  const code = input.code?.trim();
  if (!code) {
    return createActionError("Enter the 6-digit code from your authenticator app.");
  }

  const session = await getMemberSession();
  if (!session) {
    return createActionError(
      "You must be signed in to complete two-factor authentication setup."
    );
  }

  const memberId = session.member?.member_id;
  const organizationId = session.organization?.organization_id;
  if (!memberId || !organizationId) {
    return createActionError(
      "Your session is missing member or organization context."
    );
  }

  try {
    const client = getStytchB2BClient();
    const result = await client.totps.authenticate({
      organization_id: organizationId,
      member_id: memberId,
      code,
      session_jwt: session.session_jwt,
      set_mfa_enrollment: "enroll",
    });

    return createActionSuccess({
      totpRegistrationId: result.member?.totp_registration_id ?? "",
    });
  } catch (error) {
    console.error("[MFA] TOTP enrollment verification failed:", error);
    return createActionError(extractErrorMessage(error));
  }
}

/**
 * Rotate a member's recovery codes (manage-existing path).
 *
 * Invalidates all existing recovery codes and returns a fresh set — used from
 * the settings Security section when the member already holds a TOTP
 * registration (duplicate-guard surface, design D2). The new codes are shown
 * exactly once, matching the enrollment flow.
 */
export async function rotateRecoveryCodes(): Promise<
  ActionResult<{ recoveryCodes: string[] }>
> {
  const session = await getMemberSession();
  if (!session) {
    return createActionError(
      "You must be signed in to rotate recovery codes."
    );
  }

  const memberId = session.member?.member_id;
  const organizationId = session.organization?.organization_id;
  if (!memberId || !organizationId) {
    return createActionError(
      "Your session is missing member or organization context."
    );
  }

  try {
    const client = getStytchB2BClient();
    const result = await client.recoveryCodes.rotate({
      organization_id: organizationId,
      member_id: memberId,
    });

    return createActionSuccess({ recoveryCodes: result.recovery_codes });
  } catch (error) {
    console.error("[MFA] Recovery codes rotation failed:", error);
    return createActionError(extractErrorMessage(error));
  }
}

/**
 * Complete an MFA challenge at login (TOTP code or recovery code).
 *
 * TOTP path: `totps.authenticate` with the intermediate session token.
 * Recovery path: `recoveryCodes.recover` with the intermediate session token —
 * the response carries `session_token`/`session_jwt` DIRECTLY (verified
 * `B2BRecoveryCodesRecoverResponse`); the intermediate token is single-use and
 * consumed by the recover call, so there is NO chained TOTP authenticate.
 *
 * Session cookies are set ONLY on success (state-transition invariant, design
 * D1). Every attempt is rate-limited per member + per IP BEFORE any outbound
 * Stytch call; limiter rejection returns a generic error that never reveals
 * whether the code was close or valid, and records a bounded
 * `mfa_challenge_failed` audit detail (best-effort).
 */
export async function authenticateTotp(
  input: AuthenticateTotpInput
): Promise<ActionResult<{ memberAuthenticated: boolean }>> {
  const { intermediateSessionToken, code, recoveryCode, memberId, organizationId } =
    input;

  if (!intermediateSessionToken) {
    return createActionError("Your login session has expired. Please sign in again.");
  }

  const isRecovery = Boolean(recoveryCode);
  const isTotp = Boolean(code);
  if (isRecovery === isTotp) {
    return createActionError("Provide either a code or a recovery code.");
  }

  // Rate limit BEFORE any outbound Stytch call: the recovery-code path is a
  // static bearer secret and TOTP attempts are the brute-force surface; the
  // generic error below does not reveal code validity.
  const headerStore = await headers();
  const { allowed } = checkMfaRateLimit({
    memberId,
    ip: deriveClientIp(headerStore),
  });

  if (!allowed) {
    console.warn(
      "[MFA] Rate limit hit for authenticateTotp (not revealing to client):",
      { memberId }
    );
    // Bounded, non-blocking audit: detail is drawn from the closed enum and
    // never reveals code validity.
    await recordAuthAudit({
      type: "mfa_challenge_failed",
      memberId,
      organizationId,
      detail: "internal_error",
    });
    return createActionError(GENERIC_MFA_ERROR);
  }

  const duration = getSessionDurationMinutes();

  try {
    const client = getStytchB2BClient();

    if (isRecovery) {
      // Recovery path: recover() returns the full session in the same call.
      // Exclusivity is validated above (exactly one of code/recoveryCode).
      const result = await client.recoveryCodes.recover({
        organization_id: organizationId ?? "",
        member_id: memberId ?? "",
        recovery_code: recoveryCode as string,
        intermediate_session_token: intermediateSessionToken,
        session_duration_minutes: duration,
      });

      await setSessionCookies(
        result.session_token,
        result.session_jwt,
        duration
      );

      const resolvedMemberId = result.member?.member_id ?? memberId;
      const resolvedOrgId = result.organization?.organization_id ?? organizationId;

      // Best-effort audit; never alters the auth outcome.
      await recordAuthAudit({
        type: "mfa_challenge_passed",
        memberId: resolvedMemberId,
        organizationId: resolvedOrgId,
      });
      await recordAuthAudit({
        type: "login_succeeded",
        memberId: resolvedMemberId,
        organizationId: resolvedOrgId,
      });

      return createActionSuccess({ memberAuthenticated: true });
    }

    // TOTP path.
    const result = await client.totps.authenticate({
      organization_id: organizationId ?? "",
      member_id: memberId ?? "",
      code: code ?? "",
      intermediate_session_token: intermediateSessionToken,
      session_duration_minutes: duration,
    });

    await setSessionCookies(result.session_token, result.session_jwt, duration);

    const resolvedMemberId = result.member?.member_id ?? memberId;
    const resolvedOrgId = result.organization?.organization_id ?? organizationId;

    await recordAuthAudit({
      type: "mfa_challenge_passed",
      memberId: resolvedMemberId,
      organizationId: resolvedOrgId,
    });
    await recordAuthAudit({
      type: "login_succeeded",
      memberId: resolvedMemberId,
      organizationId: resolvedOrgId,
    });

    return createActionSuccess({ memberAuthenticated: true });
  } catch (error) {
    // No cookies were set (the setSessionCookies call above only runs on
    // success). Bounded audit detail — never the raw error body.
    await recordAuthAudit({
      type: "mfa_challenge_failed",
      memberId,
      organizationId,
      detail: mapAuthErrorToDetail(error),
    });
    return createActionError(extractErrorMessage(error));
  }
}
