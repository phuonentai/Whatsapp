/**
 * Server-only utilities for building auth URLs and config
 * These read from process.env and are NOT bundled into client code
 */


import "server-only";
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

export function getBaseUrl(): string {
  let baseUrl =
    process.env.APP_BASE_URL ||
    process.env.NEXT_PUBLIC_BASE_URL ||
    process.env.NEXT_PUBLIC_APP_BASE_URL ||
    process.env.APP_URL ||
    null;

  const resolved = baseUrl || "http://localhost:3000";
  return resolved.replace(/\/$/, "");
}

function resolveRoute(value: string | undefined, fallback: string): URL {
  const base = getBaseUrl();
  const trimmed = value?.trim();

  if (!trimmed) {
    return new URL(fallback, `${base}/`);
  }

  if (/^https?:\/\//i.test(trimmed)) {
    return new URL(trimmed);
  }

  const path = trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
  return new URL(path, `${base}/`);
}

export function buildLoginUrl(options?: LoginUrlOptions): string {
  const url = resolveRoute(process.env.NEXT_PUBLIC_STYTCH_LOGIN_PATH, "/auth");
  const returnTo = sanitizeReturnTo(options?.returnTo);

  if (returnTo) {
    url.searchParams.set("returnTo", returnTo);
  }

  return url.toString();
}

export function buildLogoutUrl(options?: LogoutUrlOptions): string {
  const url = resolveRoute(
    process.env.NEXT_PUBLIC_STYTCH_LOGOUT_PATH,
    "/api/auth/logout"
  );
  const returnTo = sanitizeReturnTo(options?.returnTo);

  if (returnTo) {
    url.searchParams.set("returnTo", returnTo);
  }

  return url.toString();
}

/**
 * Build complete client config from environment variables
 * This should only be called on the server
 */
export function buildStytchClientConfig(): StytchClientConfig {
  const publicToken = process.env.NEXT_PUBLIC_STYTCH_PUBLIC_TOKEN;
  if (!publicToken) {
    throw new Error("NEXT_PUBLIC_STYTCH_PUBLIC_TOKEN is required");
  }

  return {
    publicToken,
    sessionDurationMinutes:
      Number(process.env.NEXT_PUBLIC_STYTCH_SESSION_DURATION_MINUTES ?? "480") ||
      480,
    loginPath: process.env.NEXT_PUBLIC_STYTCH_LOGIN_PATH || "/auth",
    logoutPath:
      process.env.NEXT_PUBLIC_STYTCH_LOGOUT_PATH || "/api/auth/logout",
    redirectPath:
      process.env.NEXT_PUBLIC_STYTCH_REDIRECT_PATH || "/authenticate",
    baseUrl: getBaseUrl(),
  };
}
