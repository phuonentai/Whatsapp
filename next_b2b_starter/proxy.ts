import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";
import {
  SESSION_COOKIE_NAME,
  SESSION_JWT_COOKIE_NAME,
} from "@/lib/auth/constants";

// Canonical host for SEO
const CANONICAL_HOST = "yourdomain.com";

// Protected routes that require authentication
const PROTECTED_ROUTES = ["/dashboard", "/settings"];

// Public routes that never require authentication
const PUBLIC_ROUTES = ["/auth", "/authenticate", "/signup", "/api/auth"];

// ---------------------------------------------------------------------------
// Stateless Stytch session JWT validation at the edge.
//
// Stytch session JWTs are RS256-signed and verified locally against the
// project's JWKS endpoint. Keys are cached for at most JWKS_CACHE_TTL_MS
// (<= 300s, per the edge-middleware-session spec); on a fetch failure the
// cached keys keep being used while still fresh, otherwise protected routes
// fail closed with a 500. No synchronous Stytch API calls are made.
// ---------------------------------------------------------------------------
const JWKS_CACHE_TTL_MS = 300_000; // 300s — spec invariant: TTL <= 300s
const STYTCH_ORGANIZATION_CLAIM = "https://stytch.com/organization";

interface CachedJwks {
  keys: StytchJwk[];
  fetchedAt: number;
}

/** JWK shape served by the Stytch JWKS endpoint (adds kid to lib.dom's type). */
interface StytchJwk extends JsonWebKey {
  kid?: string;
}

let jwksCache: CachedJwks | null = null;

/** Resolve the Stytch base URL for the configured project environment. */
function stytchBaseUrl(): string {
  const env =
    process.env.STYTCH_PROJECT_ENV ||
    process.env.NEXT_PUBLIC_STYTCH_PROJECT_ENV ||
    "";
  if (env === "live" || env === "production") {
    return "https://api.stytch.com";
  }
  if (env === "test" || env === "development") {
    return "https://test.stytch.com";
  }
  // Fall back to the SDK convention: project-live-* means live.
  const projectId = getProjectId();
  return projectId.startsWith("project-live-")
    ? "https://api.stytch.com"
    : "https://test.stytch.com";
}

function getProjectId(): string {
  return (
    process.env.STYTCH_PROJECT_ID ||
    process.env.NEXT_PUBLIC_STYTCH_PROJECT_ID ||
    ""
  );
}

/** Fetch the project JWKS from Stytch. Throws on any failure. */
async function fetchJwksKeys(): Promise<StytchJwk[]> {
  if (!getProjectId()) {
    throw new Error("STYTCH_PROJECT_ID is not set");
  }
  const res = await fetch(
    `${stytchBaseUrl()}/v1/b2b/sessions/jwks/${encodeURIComponent(
      getProjectId()
    )}`,
    {
      headers: { accept: "application/json" },
      // Cache in the edge runtime only via our own TTL-bounded cache; never
      // rely on the platform fetch cache for key rotation.
      cache: "no-store",
    }
  );
  if (!res.ok) {
    throw new Error(`JWKS fetch failed with status ${res.status}`);
  }
  const body = (await res.json()) as { keys?: JsonWebKey[] };
  if (!Array.isArray(body.keys) || body.keys.length === 0) {
    throw new Error("JWKS response contained no keys");
  }
  return body.keys;
}

/**
 * Return the JWKS keys to validate against, or null when no usable keys exist.
 *
 * Fresh cache -> served without a fetch. Otherwise the keys are (re)fetched;
 * on failure the cached keys are still used while unexpired, and an empty or
 * expired cache yields null so protected routes can fail closed (500).
 */
export async function getJwksKeys(): Promise<StytchJwk[] | null> {
  const now = Date.now();
  if (jwksCache && now - jwksCache.fetchedAt < JWKS_CACHE_TTL_MS) {
    return jwksCache.keys;
  }
  try {
    const keys = await fetchJwksKeys();
    jwksCache = { keys, fetchedAt: Date.now() };
    return keys;
  } catch {
    // Fetch failed: keep serving cached keys while they are still valid.
    if (jwksCache && Date.now() - jwksCache.fetchedAt < JWKS_CACHE_TTL_MS) {
      return jwksCache.keys;
    }
    return null;
  }
}

function base64UrlToUint8Array(input: string): Uint8Array {
  const base64 = input.replace(/-/g, "+").replace(/_/g, "/");
  const padded = base64.padEnd(Math.ceil(base64.length / 4) * 4, "=");
  const binary = atob(padded);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}

function decodeJsonSegment(segment: string): Record<string, unknown> | null {
  try {
    return JSON.parse(
      new TextDecoder().decode(base64UrlToUint8Array(segment))
    ) as Record<string, unknown>;
  } catch {
    return null;
  }
}

function extractOrganizationId(payload: Record<string, unknown>): string {
  const orgClaim = payload[STYTCH_ORGANIZATION_CLAIM];
  if (typeof orgClaim === "object" && orgClaim !== null) {
    // Narrowed object; read the namespaced organization_id field.
    const orgClaimRecord = orgClaim as Record<string, unknown>;
    if (typeof orgClaimRecord.organization_id === "string") {
      return orgClaimRecord.organization_id;
    }
  }
  if (typeof payload.organization_id === "string") {
    return payload.organization_id;
  }
  if (typeof payload.org_id === "string") {
    return payload.org_id;
  }
  return "";
}

/**
 * Statelessly verify a Stytch B2B session JWT against the project JWKS.
 *
 * Returns the member and organization ids on success, or null when the token
 * is malformed, expired, signed by an unknown key, or missing claims.
 */
export async function verifySessionJwt(
  jwt: string,
  keys: StytchJwk[]
): Promise<{ memberId: string; organizationId: string } | null> {
  const parts = jwt.split(".");
  if (parts.length !== 3) {
    return null;
  }
  const [headerSegment, payloadSegment, signatureSegment] = parts;
  const header = decodeJsonSegment(headerSegment);
  const payload = decodeJsonSegment(payloadSegment);
  if (!header || !payload) {
    return null;
  }

  // Stytch signs session JWTs with RS256 only.
  if (header.alg !== "RS256") {
    return null;
  }

  // Reject tokens not issued for this project (matches the SDK's issuer check).
  const projectId = getProjectId();
  const allowedIssuers = [
    `stytch.com/${projectId}`,
    stytchBaseUrl().replace(/\/$/, ""),
  ];
  if (
    typeof payload.iss === "string" &&
    !allowedIssuers.includes(payload.iss)
  ) {
    return null;
  }

  // exp is a Unix timestamp in seconds; missing or elapsed means expired.
  if (typeof payload.exp !== "number" || payload.exp * 1000 <= Date.now()) {
    return null;
  }

  const kid = typeof header.kid === "string" ? header.kid : "";
  const key = keys.find(
    (k) =>
      k.kty === "RSA" &&
      k.use !== "enc" &&
      (!kid || k.kid === kid)
  );
  if (!key) {
    return null;
  }

  const cryptoKey = await crypto.subtle.importKey(
    "jwk",
    key,
    { name: "RSASSA-PKCS1-v1_5", hash: "SHA-256" },
    false,
    ["verify"]
  );
  const data = new TextEncoder().encode(`${headerSegment}.${payloadSegment}`);
  const signature = base64UrlToUint8Array(signatureSegment);
  const valid = await crypto.subtle.verify(
    "RSASSA-PKCS1-v1_5",
    cryptoKey,
    signature,
    data
  );
  if (!valid) {
    return null;
  }

  const memberId = typeof payload.sub === "string" ? payload.sub : "";
  const organizationId = extractOrganizationId(payload);
  if (memberId === "" || organizationId === "") {
    return null;
  }

  return { memberId, organizationId };
}

/** Redirect to /auth, clearing any session cookies, with the path as returnTo. */
function redirectToAuth(request: NextRequest, pathname: string): NextResponse {
  const loginUrl = new URL("/auth", request.url);
  loginUrl.searchParams.set("returnTo", pathname);
  const response = NextResponse.redirect(loginUrl, 302);
  // Clear both session cookies so a stale/invalid session cannot loop.
  for (const name of [SESSION_COOKIE_NAME, SESSION_JWT_COOKIE_NAME]) {
    response.cookies.set(name, "", { path: "/", maxAge: 0 });
  }
  return response;
}

export async function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const url = request.nextUrl.clone();
  const host = request.headers.get("host") || "";

  // Skip redirects for local/dev hosts
  const isLocalhost =
    host.startsWith("localhost") || host.startsWith("127.0.0.1");

  // 1. SEO: Force apex domain (remove www.) - skip for localhost/dev
  if (!isLocalhost && host === `www.${CANONICAL_HOST}`) {
    url.hostname = CANONICAL_HOST;
    return NextResponse.redirect(url, 301);
  }

  // 2. SEO: Enforce HTTPS on canonical domain
  if (!isLocalhost && host === CANONICAL_HOST) {
    const proto = request.headers.get("x-forwarded-proto");
    if (proto && proto !== "https") {
      url.protocol = "https:";
      return NextResponse.redirect(url, 301);
    }
  }

  // 3. Check if route is public (no auth needed)
  const isPublicRoute = PUBLIC_ROUTES.some((route) =>
    pathname.startsWith(route)
  );
  if (isPublicRoute) {
    return NextResponse.next();
  }

  // 4. Check if route is protected (auth required)
  const isProtectedRoute = PROTECTED_ROUTES.some((route) =>
    pathname.startsWith(route)
  );
  if (!isProtectedRoute) {
    return NextResponse.next();
  }

  // Mock auth mode (test/dev only): the X-Test-Org-ID cookie/header grants
  // access. The mock context is ALSO injected as a cookie into the downstream
  // request (and set on the response) so server components, route handlers,
  // and rewrites all see it on the very first navigation. This branch runs
  // ahead of real validation so mock sessions keep working when enabled.
  if (process.env.AUTH_MOCK_ENABLED === "true") {
    const mockOrg =
      request.cookies.get("X-Test-Org-ID")?.value ??
      request.headers.get("X-Test-Org-ID") ??
      "";
    if (mockOrg) {
      const injectedCookie =
        `X-Test-Org-ID=${encodeURIComponent(mockOrg)}` +
        (request.headers.get("cookie")
          ? `; ${request.headers.get("cookie")}`
          : "");
      const response = NextResponse.next({
        request: {
          headers: new Headers([
            ...request.headers.entries(),
            ["cookie", injectedCookie],
          ]),
        },
      });
      response.cookies.set("X-Test-Org-ID", mockOrg, {
        path: "/",
        maxAge: 3600,
        sameSite: "lax",
      });
      return response;
    }
  }

  // 5. Stateless JWT validation (presence-only checks are prohibited).
  const sessionJwt = request.cookies.get(SESSION_JWT_COOKIE_NAME)?.value;

  // Missing JWT -> clear any session cookies and redirect to /auth.
  if (!sessionJwt) {
    return redirectToAuth(request, pathname);
  }

  // No usable JWKS keys (empty/expired cache and fetch failed) -> fail closed.
  const keys = await getJwksKeys();
  if (!keys) {
    return new NextResponse("Service Unavailable", { status: 500 });
  }

  const claims = await verifySessionJwt(sessionJwt, keys);

  // Invalid/expired JWT -> clear the session cookies and redirect to /auth.
  if (!claims) {
    return redirectToAuth(request, pathname);
  }

  // Valid JWT: forward the validated identity. Client-supplied forwarded-auth
  // headers are NEVER trusted — strip them and set only the proxy's own.
  const requestHeaders = new Headers(request.headers);
  requestHeaders.delete("X-Forwarded-Auth");
  requestHeaders.delete("X-Stytch-Organization-Id");
  requestHeaders.delete("X-Stytch-Member-Id");
  requestHeaders.set("X-Forwarded-Auth", "true");
  requestHeaders.set("X-Stytch-Organization-Id", claims.organizationId);
  requestHeaders.set("X-Stytch-Member-Id", claims.memberId);

  return NextResponse.next({ request: { headers: requestHeaders } });
}

export const config = {
  matcher: [
    /*
     * Match all request paths except:
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     * - public files (images, etc.)
     */
    "/((?!_next/static|_next/image|favicon.ico|icon.png|.*\\.(?:svg|png|jpg|jpeg|gif|webp)$).*)",
  ],
};
