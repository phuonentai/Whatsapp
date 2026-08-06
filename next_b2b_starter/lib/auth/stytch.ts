// Authentication URL builders for Stytch
// These functions handle URL construction with proper safety checks

export interface LoginUrlOptions {
  returnTo?: string;
}

export interface LogoutUrlOptions {
  returnTo?: string;
}

export function sanitizeReturnTo(value: string | undefined): string | undefined {
  if (!value) return undefined;
  const trimmed = value.trim();
  if (!trimmed.startsWith("/") || trimmed.startsWith("//")) {
    return undefined;
  }
  return trimmed;
}

export function getBaseUrl(): string {
  if (typeof window !== "undefined") {
    return window.location.origin.replace(/\/$/, "");
  }

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
  const url = resolveRoute(process.env.NEXT_PUBLIC_STYTCH_LOGOUT_PATH, "/api/auth/logout");
  const returnTo = sanitizeReturnTo(options?.returnTo);

  if (returnTo) {
    url.searchParams.set("returnTo", returnTo);
  }

  return url.toString();
}
