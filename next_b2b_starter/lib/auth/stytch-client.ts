/**
 * Client-safe utilities that use passed config instead of reading env
 * These functions are pure and can be safely used in client components
 */

import type { StytchClientConfig } from "./config-types";

export interface LoginUrlOptions {
  returnTo?: string;
}

export interface LogoutUrlOptions {
  returnTo?: string;
}

function sanitizeReturnTo(value: string | undefined): string | undefined {
  if (!value) return undefined;
  const trimmed = value.trim();
  if (!trimmed.startsWith("/") || trimmed.startsWith("//")) {
    return undefined;
  }
  return trimmed;
}

function resolveRoute(
  baseUrl: string,
  path: string,
  fallback: string
): URL {
  const trimmed = path?.trim();

  if (!trimmed) {
    return new URL(fallback, `${baseUrl}/`);
  }

  if (/^https?:\/\//i.test(trimmed)) {
    return new URL(trimmed);
  }

  const cleanPath = trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
  return new URL(cleanPath, `${baseUrl}/`);
}

export function buildLoginUrl(
  config: StytchClientConfig,
  options?: LoginUrlOptions
): string {
  const url = resolveRoute(config.baseUrl, config.loginPath, "/auth");
  const returnTo = sanitizeReturnTo(options?.returnTo);

  if (returnTo) {
    url.searchParams.set("returnTo", returnTo);
  }

  return url.toString();
}

export function buildLogoutUrl(
  config: StytchClientConfig,
  options?: LogoutUrlOptions
): string {
  const url = resolveRoute(
    config.baseUrl,
    config.logoutPath,
    "/api/auth/logout"
  );
  const returnTo = sanitizeReturnTo(options?.returnTo);

  if (returnTo) {
    url.searchParams.set("returnTo", returnTo);
  }

  return url.toString();
}
