
import type { Member, MemberSession } from "stytch";

import {
  buildLoginUrl as buildStytchLoginUrl,
  buildLogoutUrl as buildStytchLogoutUrl,
  getBaseUrl as getStytchBaseUrl,
} from "@/lib/auth/stytch-server";

const SUBSCRIPTION_CLAIM = "https://api.yourdomain.com/subscription";
const SUBSCRIPTION_FLAG = "https://api.yourdomain.com/has_active_subscription";

/**
 * Check if a user has an active subscription
 * This function is safe for both server and client contexts
 */
export function hasActiveSubscription(
  session: Pick<MemberSession, "custom_claims" | "roles"> | undefined | null,
  member?: Pick<Member, "roles" | "trusted_metadata"> | null
): boolean {
  if (!session && !member) return false;

  const customClaims = session?.custom_claims ?? {};
  const trustedMetadata = (member?.trusted_metadata ?? {}) as Record<
    string,
    unknown
  >;

  const status =
    (customClaims[SUBSCRIPTION_CLAIM] as string | undefined) ||
    (trustedMetadata["subscription_status"] as string | undefined);

  if (status) {
    const normalized = status.toLowerCase();
    if (["active", "trialing", "grace"].includes(normalized)) return true;
    if (["canceled", "cancelled", "inactive", "past_due"].includes(normalized))
      return false;
  }

  const roles = new Set<string>();
  session?.roles?.forEach((role) => roles.add(role));
  member?.roles?.forEach((role) => {
    if (role?.role_id) {
      roles.add(role.role_id);
    }
  });

  if (roles.has("subscriber")) return true;

  const hasFlag =
    (customClaims[SUBSCRIPTION_FLAG] as boolean | undefined) ??
    (trustedMetadata["has_active_subscription"] as boolean | undefined);

  if (typeof hasFlag === "boolean") {
    return hasFlag;
  }

  return true;
}

/**
 * Get base URL - server-only function
 * @throws Error if called on client
 */
export function getBaseUrl(): string {
  if (typeof window !== "undefined") {
    throw new Error("getBaseUrl() must only be called on server");
  }

  return getStytchBaseUrl();
}

export interface LoginUrlOptions {
  returnTo?: string;
}

/**
 * Build login URL - server-only function
 * @throws Error if called on client
 */
export function buildLoginUrl(options?: LoginUrlOptions) {
  return buildStytchLoginUrl({
    returnTo: options?.returnTo,
  });
}

export interface LogoutUrlOptions {
  returnTo?: string;
}

/**
 * Build logout URL - server-only function
 * @throws Error if called on client
 */
export function buildLogoutUrl(options?: LogoutUrlOptions) {
  return buildStytchLogoutUrl({
    returnTo: options?.returnTo,
  });
}
