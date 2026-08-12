import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import type { NextRequest } from "next/server";
import { webcrypto } from "node:crypto";

// jsdom's window.crypto has no subtle; fall back to Node's WebCrypto so we can
// mint and verify real RS256 tokens.
if (!globalThis.crypto?.subtle) {
  Object.defineProperty(globalThis, "crypto", {
    value: webcrypto,
    configurable: true,
  });
}

// Mock NextResponse so tests can inspect redirect targets, forwarded request
// headers, and cookie clear operations without pulling in the Next runtime.
const { MockNextResponse } = vi.hoisted(() => {
  class MockNextResponse {
    status: number;
    headers: Headers;
    requestHeaders: Headers | null;
    cookieSets: Array<{
      name: string;
      value: string;
      options?: Record<string, unknown>;
    }>;
    cookies: {
      set: (
        name: string,
        value: string,
        options?: Record<string, unknown>
      ) => void;
      delete: (name: string) => void;
    };

    constructor(body?: BodyInit | null, init?: ResponseInit) {
      this.status = init?.status ?? 200;
      this.headers = new Headers(init?.headers);
      this.requestHeaders = null;
      this.cookieSets = [];
      this.cookies = {
        set: (
          name: string,
          value: string,
          options?: Record<string, unknown>
        ) => {
          this.cookieSets.push({ name, value, options });
        },
        delete: (name: string) => {
          this.cookieSets.push({ name, value: "", options: { maxAge: 0 } });
        },
      };
    }

    static next(opts?: { request?: { headers?: Headers } }) {
      const res = new MockNextResponse();
      res.requestHeaders = opts?.request?.headers ?? null;
      return res;
    }

    static redirect(url: string | URL, status = 302) {
      const res = new MockNextResponse();
      res.status = status;
      res.headers.set("location", url.toString());
      return res;
    }
  }

  return { MockNextResponse };
});

// Instance type of the mocked response, used to type runProxy's return value.
type MockResponse = InstanceType<typeof MockNextResponse>;

vi.mock("next/server", () => ({ NextResponse: MockNextResponse }));

const SESSION_JWT = "stytch_session_jwt";
const SESSION_TOKEN = "stytch_session";
const ORG_ID = "550e8400-e29b-41d4-a716-446655440000";
const MEMBER_ID = "660e8400-e29b-41d4-a716-446655440001";
const TEST_ISSUER = "https://test.stytch.com";

let privateKey: CryptoKey;
type TestJwk = JsonWebKey & { kid: string };
let publicJwk: TestJwk;
const KID = "test-kid-1";

function base64UrlEncode(input: string | Uint8Array): string {
  const bytes =
    typeof input === "string" ? new TextEncoder().encode(input) : input;
  let binary = "";
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

async function signJwt(
  payload: Record<string, unknown>,
  key: CryptoKey = privateKey
): Promise<string> {
  const headerSegment = base64UrlEncode(
    JSON.stringify({ alg: "RS256", kid: KID, typ: "JWT" })
  );
  const payloadSegment = base64UrlEncode(JSON.stringify(payload));
  const data = new TextEncoder().encode(`${headerSegment}.${payloadSegment}`);
  const signature = await crypto.subtle.sign("RSASSA-PKCS1-v1_5", key, data);
  return `${headerSegment}.${payloadSegment}.${base64UrlEncode(
    new Uint8Array(signature)
  )}`;
}

function validPayload(overrides: Record<string, unknown> = {}) {
  return {
    sub: MEMBER_ID,
    iss: TEST_ISSUER,
    iat: Math.floor(Date.now() / 1000) - 60,
    exp: Math.floor(Date.now() / 1000) + 3600,
    "https://stytch.com/organization": { organization_id: ORG_ID },
    "https://stytch.com/session": {
      id: "session-test-1",
      started_at: new Date().toISOString(),
      last_accessed_at: new Date().toISOString(),
      expires_at: new Date(Date.now() + 3600_000).toISOString(),
      authentication_factors: [],
      roles: ["stytch_member"],
    },
    ...overrides,
  };
}

function makeRequest(
  path: string,
  opts: {
    cookies?: Record<string, string>;
    headers?: Record<string, string>;
    host?: string;
  } = {}
): NextRequest {
  const nextUrl = new URL(path, "http://localhost:3000");
  // NextURL (used by NextRequest in the edge runtime) exposes clone(); a plain
  // WHATWG URL does not. Provide it so the SEO block runs unchanged in tests.
  const nextUrlWithClone = nextUrl as unknown as { clone: () => URL };
  nextUrlWithClone.clone = () => nextUrl;
  const headers = new Headers(opts.headers ?? {});
  if (opts.host) {
    headers.set("host", opts.host);
  }
  return {
    nextUrl,
    url: nextUrl.toString(),
    headers,
    cookies: {
      get: (name: string) =>
        opts.cookies && name in opts.cookies
          ? { name, value: opts.cookies[name] }
          : undefined,
    },
  } as unknown as NextRequest;
}

function stubJwksFetch(keys: TestJwk[], ok = true) {
  const fetchMock = vi.fn(async () => {
    if (!ok) {
      return new Response("boom", { status: 500 });
    }
    return new Response(JSON.stringify({ keys }), {
      status: 200,
      headers: { "content-type": "application/json" },
    });
  });
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

// Dynamic import + resetModules is intentional (module loading boundary):
// proxy.ts keeps module-level JWKS cache state that must be reset between
// tests, so a static import would leak cache across cases.
async function loadProxy() {
  vi.resetModules();
  return await import("./proxy");
}

// Test seam: run the proxy and view the result through the mock's lens so
// tests can assert on forwarded request headers and cookie-clear operations.
async function runProxy(
  proxyFn: (request: NextRequest) => Promise<unknown>,
  request: NextRequest
): Promise<MockResponse> {
  return (await proxyFn(request)) as unknown as MockResponse;
}

describe("proxy.ts - stateless Stytch JWT validation", () => {
  beforeAll(async () => {
    const pair = await crypto.subtle.generateKey(
      {
        name: "RSASSA-PKCS1-v1_5",
        modulusLength: 2048,
        publicExponent: new Uint8Array([1, 0, 1]),
        hash: "SHA-256",
      },
      true,
      ["sign", "verify"]
    );
    privateKey = pair.privateKey;
    publicJwk = {
      ...(await crypto.subtle.exportKey("jwk", pair.publicKey)),
      kid: KID,
      use: "sig",
    };
  });

  beforeEach(() => {
    process.env.STYTCH_PROJECT_ID = "project-test-123";
    process.env.STYTCH_PROJECT_ENV = "test";
    delete process.env.AUTH_MOCK_ENABLED;
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    delete process.env.STYTCH_PROJECT_ID;
    delete process.env.STYTCH_PROJECT_ENV;
    delete process.env.AUTH_MOCK_ENABLED;
  });

  it("forwards validated identity headers and allows a valid JWT on a protected route", async () => {
    const { proxy } = await loadProxy();
    stubJwksFetch([publicJwk]);
    const jwt = await signJwt(validPayload());

    const response = await runProxy(proxy, 
      makeRequest("/dashboard", { cookies: { [SESSION_JWT]: jwt } })
    );

    expect(response.status).toBe(200);
    expect(response.requestHeaders?.get("X-Forwarded-Auth")).toBe("true");
    expect(response.requestHeaders?.get("X-Stytch-Organization-Id")).toBe(
      ORG_ID
    );
    expect(response.requestHeaders?.get("X-Stytch-Member-Id")).toBe(MEMBER_ID);
  });

  it("strips client-supplied forwarded-auth headers before emitting its own", async () => {
    const { proxy } = await loadProxy();
    stubJwksFetch([publicJwk]);
    const jwt = await signJwt(validPayload());

    const response = await runProxy(proxy, 
      makeRequest("/settings/team", {
        cookies: { [SESSION_JWT]: jwt },
        headers: {
          "X-Forwarded-Auth": "true",
          "X-Stytch-Organization-Id": "attacker-org",
          "X-Stytch-Member-Id": "attacker-member",
        },
      })
    );

    expect(response.status).toBe(200);
    expect(response.requestHeaders?.get("X-Forwarded-Auth")).toBe("true");
    expect(response.requestHeaders?.get("X-Stytch-Organization-Id")).toBe(
      ORG_ID
    );
    expect(response.requestHeaders?.get("X-Stytch-Member-Id")).toBe(MEMBER_ID);
  });

  it("redirects to /auth with returnTo and clears cookies when the JWT is missing", async () => {
    const { proxy } = await loadProxy();
    const fetchMock = stubJwksFetch([publicJwk]);

    const response = await runProxy(proxy, makeRequest("/dashboard/crm"));

    expect(response.status).toBe(302);
    const location = new URL(response.headers.get("location") ?? "");
    expect(location.pathname).toBe("/auth");
    expect(location.searchParams.get("returnTo")).toBe("/dashboard/crm");
    // No JWT to validate -> no JWKS fetch performed.
    expect(fetchMock).not.toHaveBeenCalled();
    expect(
      response.cookieSets.filter((c) => c.value === "").map((c) => c.name)
    ).toEqual([SESSION_TOKEN, SESSION_JWT]);
  });

  it("redirects to /auth and clears cookies for an expired JWT", async () => {
    const { proxy } = await loadProxy();
    stubJwksFetch([publicJwk]);
    const jwt = await signJwt(
      validPayload({ exp: Math.floor(Date.now() / 1000) - 60 })
    );

    const response = await runProxy(proxy, 
      makeRequest("/dashboard", { cookies: { [SESSION_JWT]: jwt } })
    );

    expect(response.status).toBe(302);
    expect(response.headers.get("location")).toContain("/auth");
    expect(
      response.cookieSets.filter((c) => c.value === "").map((c) => c.name)
    ).toEqual([SESSION_TOKEN, SESSION_JWT]);
  });

  it("redirects to /auth and clears cookies for an invalid (tampered) JWT", async () => {
    const { proxy } = await loadProxy();
    stubJwksFetch([publicJwk]);
    const jwt = await signJwt(validPayload());
    // Flip a character in the payload segment so the signature no longer matches.
    const parts = jwt.split(".");
    const tampered = `${parts[0]}.${parts[1].slice(0, -2)}xx.${parts[2]}`;

    const response = await runProxy(proxy, 
      makeRequest("/dashboard", { cookies: { [SESSION_JWT]: tampered } })
    );

    expect(response.status).toBe(302);
    expect(response.headers.get("location")).toContain("/auth");
    expect(
      response.cookieSets.filter((c) => c.value === "").map((c) => c.name)
    ).toEqual([SESSION_TOKEN, SESSION_JWT]);
  });

  it("redirects to /auth and clears cookies for a malformed JWT", async () => {
    const { proxy } = await loadProxy();
    stubJwksFetch([publicJwk]);

    const response = await runProxy(proxy, 
      makeRequest("/dashboard", { cookies: { [SESSION_JWT]: "not-a-jwt" } })
    );

    expect(response.status).toBe(302);
    expect(response.headers.get("location")).toContain("/auth");
    expect(
      response.cookieSets.filter((c) => c.value === "").map((c) => c.name)
    ).toEqual([SESSION_TOKEN, SESSION_JWT]);
  });

  it("redirects to /auth and clears cookies for a JWT signed by an unknown key", async () => {
    const { proxy } = await loadProxy();
    // Serve a JWKS that does not contain the signing key.
    stubJwksFetch([{ ...publicJwk, kid: "other-key" }]);
    const jwt = await signJwt(validPayload());

    const response = await runProxy(proxy, 
      makeRequest("/dashboard", { cookies: { [SESSION_JWT]: jwt } })
    );

    expect(response.status).toBe(302);
    expect(response.headers.get("location")).toContain("/auth");
  });

  it("fails closed with 500 when JWKS is unavailable and no cache exists", async () => {
    const { proxy } = await loadProxy();
    stubJwksFetch([], false);
    const jwt = await signJwt(validPayload());

    const response = await runProxy(proxy, 
      makeRequest("/dashboard", { cookies: { [SESSION_JWT]: jwt } })
    );

    expect(response.status).toBe(500);
  });

  it("passes public routes through without validation", async () => {
    const { proxy } = await loadProxy();
    const fetchMock = stubJwksFetch([publicJwk]);

    for (const path of ["/auth", "/authenticate", "/signup", "/api/auth/session/refresh"]) {
      const response = await runProxy(proxy, makeRequest(path));
      expect(response.status).toBe(200);
      expect(response.requestHeaders).toBeNull();
    }
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("passes non-protected routes through without validation", async () => {
    const { proxy } = await loadProxy();
    const fetchMock = stubJwksFetch([publicJwk]);

    const response = await runProxy(proxy, makeRequest("/"));

    expect(response.status).toBe(200);
    expect(response.requestHeaders).toBeNull();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("grants access via the mock branch when AUTH_MOCK_ENABLED is set", async () => {
    process.env.AUTH_MOCK_ENABLED = "true";
    const { proxy } = await loadProxy();
    const fetchMock = stubJwksFetch([publicJwk]);

    const response = await runProxy(proxy, 
      makeRequest("/dashboard", {
        cookies: { "X-Test-Org-ID": "org-1:admin@test.com" },
      })
    );

    expect(response.status).toBe(200);
    // Mock branch bypasses JWT validation entirely.
    expect(fetchMock).not.toHaveBeenCalled();
  });
});
