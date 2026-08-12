"use server";

import { cookies } from "next/headers";
import { getStytchB2BClient } from "@/lib/auth/stytch/server";
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
import {
  createActionError,
  createActionSuccess,
  type ActionResult,
} from "@/lib/utils/server-action-helpers";

export interface ConsumeMagicLinkResult {
  memberAuthenticated: boolean;
  intermediateSessionToken?: string;
  member?: {
    member_id: string;
    email_address: string;
    name: string;
  };
  organization?: {
    organization_id: string;
    organization_name: string;
  };
  mfaRequired?: unknown;
  primaryRequired?: unknown;
}

/**
 * Consume Magic Link Server Action
 *
 * Exchanges a magic link token for a session.
 * Sets session cookies on successful authentication.
 *
 * @param token - The magic link token from the URL
 * @param sessionDurationMinutes - Optional session duration (defaults to env config)
 * @returns ActionResult with authentication status
 */
export async function consumeMagicLink(
  token: string,
  sessionDurationMinutes?: number
): Promise<ActionResult<ConsumeMagicLinkResult>> {
  if (!token) {
    return createActionError("Magic link token is required.");
  }

  const duration = sessionDurationMinutes || getSessionDurationMinutes();

  try {
    const client = getStytchB2BClient();

    const result = await client.magicLinks.authenticate({
      magic_links_token: token,
      session_duration_minutes: duration,
    });

    if (!result.member_authenticated) {
      // MFA challenge not yet passed — record the gated attempt. Best-effort
      // and non-blocking; the audit helper never throws.
      if (result.mfa_required) {
        await recordAuthAudit({
          type: "mfa_challenge_failed",
          memberId: result.member?.member_id,
          organizationId: result.organization?.organization_id,
          detail: "mfa_required",
        });
      }

      return createActionSuccess({
        memberAuthenticated: false,
        intermediateSessionToken: result.intermediate_session_token,
        member: result.member
          ? {
              member_id: result.member.member_id,
              email_address: result.member.email_address,
              name: result.member.name,
            }
          : undefined,
        organization: result.organization
          ? {
              organization_id: result.organization.organization_id,
              organization_name: result.organization.organization_name,
            }
          : undefined,
        mfaRequired: result.mfa_required ?? false,
        primaryRequired: result.primary_required ?? false,
      });
    }

    // Set session cookies
    const cookieStore = await cookies();
    const maxAgeSeconds = duration * 60;

    if (result.session_token) {
      cookieStore.set(SESSION_COOKIE_NAME, result.session_token, {
        ...getSecureCookieConfig(),
        maxAge: maxAgeSeconds,
      });
    }

    if (result.session_jwt) {
      cookieStore.set(SESSION_JWT_COOKIE_NAME, result.session_jwt, {
        ...getCookieConfig(),
        maxAge: maxAgeSeconds,
      });
    }

    // Record the successful login (best-effort; never blocks the auth outcome).
    // The session JWT was just set above, so the audit helper can authenticate
    // its POST to the Go activity endpoint.
    await recordAuthAudit({
      type: "login_succeeded",
      memberId: result.member?.member_id,
      organizationId: result.organization?.organization_id,
    });

    return createActionSuccess({
      memberAuthenticated: true,
      member: result.member
        ? {
            member_id: result.member.member_id,
            email_address: result.member.email_address,
            name: result.member.name,
          }
        : undefined,
      organization: result.organization
        ? {
            organization_id: result.organization.organization_id,
            organization_name: result.organization.organization_name,
          }
        : undefined,
    });
  } catch (error: unknown) {
    const errorMessage = extractErrorMessage(error);

    // Record the rejected token consumption (best-effort, non-blocking). Only
    // a bounded error code is recorded — never the raw error body.
    await recordAuthAudit({
      type: "login_failed",
      detail: mapAuthErrorToDetail(error),
    });

    return createActionError(errorMessage);
  }
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
  return "Unable to verify magic link.";
}
