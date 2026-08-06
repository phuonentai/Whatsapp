/**
 * Type-safe configuration for Stytch authentication
 * These types define the shape of config passed from server to client
 */

export interface StytchClientConfig {
  publicToken: string;
  sessionDurationMinutes: number;
  loginPath: string;
  logoutPath: string;
  redirectPath: string;
  baseUrl: string;
}

export interface CookieConfiguration {
  sessionCookieName: string;
  sessionJwtCookieName: string;
  httpOnly: boolean;
  sameSite: "lax" | "strict" | "none";
  secure: boolean;
  path: string;
}
