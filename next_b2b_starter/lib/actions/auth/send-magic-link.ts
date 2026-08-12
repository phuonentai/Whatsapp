"use server";

import { headers } from "next/headers";
import {
  getStytchB2BClient,
  getOrganizationIdsForMemberSearch,
} from "@/lib/auth/stytch/server";
import { recordAuthAudit } from "@/lib/auth/audit";
import { checkMagicLinkRateLimit } from "@/lib/auth/magic-link-limiter";
import {
  createActionSuccess,
  createActionError,
  type ActionResult,
} from "@/lib/utils/server-action-helpers";

const NEUTRAL_SEND_MESSAGE =
  "If an account exists with that email, a magic link has been sent.";

/**
 * Derive the client IP from proxy headers.
 *
 * Trust order: `x-forwarded-for` first entry -> `x-real-ip` -> localhost
 * fallback. In production the frontend sits behind a known ingress that sets
 * these headers; in local dev the fallback keeps the action usable.
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

/**
 * Send Magic Link Server Action
 *
 * Validates that an email belongs to an existing member before sending a magic link.
 * This prevents unknown users from receiving authentication emails.
 *
 * @param email - The email address to send the magic link to
 * @returns ActionResult with success message or error
 */
export async function sendMagicLink(
  email: string
): Promise<ActionResult<{ message: string }>> {
  try {
    // Validate input
    if (!email || typeof email !== "string") {
      return createActionError("Email address is required");
    }

    // Rate limit BEFORE any outbound Stytch call: throttle hit => no Stytch
    // call => neutral success-shaped response (no enumeration leak).
    const headerStore = await headers();
    const { allowed } = checkMagicLinkRateLimit({
      email,
      ip: deriveClientIp(headerStore),
    });

    if (!allowed) {
      console.info(
        "[Magic Link] Rate limit hit, suppressing send (not revealing to client):",
        email
      );
      return createActionSuccess(
        { message: NEUTRAL_SEND_MESSAGE },
        { throttled: true }
      );
    }

    const client = getStytchB2BClient();
    const organizationIds = await getOrganizationIdsForMemberSearch();

    if (!organizationIds.length) {
      console.error(
        "[Magic Link] No organization IDs configured for member search."
      );
      return createActionError(
        "Unable to process request. Please try again later."
      );
    }

    // Search for members with this email across all organizations
    // This checks if the user exists in ANY organization
    const searchResult = await client.organizations.members.search({
      organization_ids: organizationIds,
      query: {
        operator: "AND",
        operands: [
          {
            filter_name: "member_emails",
            filter_value: [email.toLowerCase()],
          },
        ],
      },
    });

    // If no members found, reject without revealing this fact
    if (!searchResult.members || searchResult.members.length === 0) {
      // Return success to prevent user enumeration
      // But don't actually send an email
      console.info(
        "[Magic Link] No member found for email (not revealing to client):",
        email
      );
      return createActionSuccess({ message: NEUTRAL_SEND_MESSAGE });
    }

    // Member exists - prepare login redirect URL
    const redirectUrl = process.env.NEXT_PUBLIC_APP_BASE_URL
      ? `${process.env.NEXT_PUBLIC_APP_BASE_URL}/authenticate`
      : "http://localhost:3000/authenticate";

    const memberOrganizationIds = Array.from(
      new Set(
        (searchResult.members ?? [])
          .map((member) => member.organization_id)
          .filter((orgId): orgId is string => Boolean(orgId))
      )
    );

    if (memberOrganizationIds.length === 0) {
      console.warn(
        "[Magic Link] Member search succeeded but no organization IDs were returned for email:",
        email
      );
      return createActionSuccess({ message: NEUTRAL_SEND_MESSAGE });
    }

    if (memberOrganizationIds.length > 1) {
      console.warn(
        "[Magic Link] Email is associated with multiple organizations; issuing login link for all memberships.",
        {
          email,
          organizationIds: memberOrganizationIds,
        }
      );
    }

    // Send magic link for each organization the member belongs to
    await Promise.all(
      memberOrganizationIds.map((organizationId) =>
        client.magicLinks.email.loginOrSignup({
          email_address: email,
          organization_id: organizationId,
          login_redirect_url: redirectUrl,
        })
      )
    );

    console.info("[Magic Link] Successfully sent magic link to:", email);

    // Record one audit row per organization the link was sent to
    // (design D4 — matches how the send loops per org). Best-effort: the
    // helper never throws, so the action outcome is unaffected.
    const memberId = searchResult.members[0]?.member_id;
    for (const organizationId of memberOrganizationIds) {
      await recordAuthAudit({
        type: "magic_link_requested",
        memberId,
        organizationId,
      });
    }

    return createActionSuccess({ message: NEUTRAL_SEND_MESSAGE });
  } catch (error: any) {
    console.error("[Magic Link] Error sending magic link:", error);

    // Return generic error to prevent user enumeration
    return createActionError(
      "Unable to process request. Please try again later.",
      process.env.NODE_ENV === "development" ? error.message : undefined
    );
  }
}
