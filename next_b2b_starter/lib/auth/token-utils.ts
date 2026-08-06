import { jwtDecode, type JwtPayload } from "jwt-decode";



export type AccessTokenPayload = JwtPayload & {
  [key: string]: unknown;
};

export function decodeAccessToken(
  token: string | null
): AccessTokenPayload | null {
  if (!token) {
    return null;
  }

  // Validate JWT structure (must have 3 parts separated by dots)
  const parts = token.split(".");
  if (parts.length !== 3) {
    console.warn("[Token] Invalid JWT structure: expected 3 parts, got", parts.length);
    return null;
  }

  try {
    const decoded = jwtDecode<AccessTokenPayload>(token);

    // Validate required JWT claims
    if (!decoded.exp || !decoded.iat) {
      console.warn("[Token] Missing required claims (exp, iat)");
      return null;
    }

    return decoded;
  } catch (error) {
    console.warn("[Token] Failed to decode JWT:", error instanceof Error ? error.message : "Unknown error");
    return null;
  }
}

export function isTokenExpired(
  tokenOrPayload: string | AccessTokenPayload | null,
  graceSeconds: number = 60
): boolean {
  const payload =
    typeof tokenOrPayload === "string"
      ? decodeAccessToken(tokenOrPayload)
      : tokenOrPayload;

  if (!payload || typeof payload.exp !== "number") {
    return true;
  }

  const nowSeconds = Math.floor(Date.now() / 1000);
  const expiresAt = payload.exp;
  const isExpired = expiresAt <= nowSeconds + graceSeconds;

  // Log if token is close to expiring for debugging
  if (isExpired) {
    const timeToExpiry = expiresAt - nowSeconds;
    console.debug("[Token] Token expired or expiring soon:", {
      expiresAt: new Date(expiresAt * 1000).toISOString(),
      timeToExpiry: `${timeToExpiry}s`,
      graceSeconds,
    });
  }

  return isExpired;
}
