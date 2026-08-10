// lib/api/client/api-client.ts

import {
  SESSION_COOKIE_NAME,
  SESSION_JWT_COOKIE_NAME,
  AUTH_ROUTES,
} from "@/lib/auth/constants";
import {
  AccessTokenPayload,
  decodeAccessToken,
  isTokenExpired,
} from "@/lib/auth/token-utils";
import { apiLogger } from "@/lib/utils/api-logger";

export interface ApiClientConfig {
  baseUrl: string;
  defaultHeaders: Record<string, string>;
}

export interface RequestOptions {
  headers?: Record<string, string>;
  skipAuth?: boolean;
  params?: Record<string, string | number | undefined>;
}

export class ApiClient {
  private config: ApiClientConfig;

  constructor(config?: Partial<ApiClientConfig>) {
    // Server-side: use internal Docker network URL (Node.js needs absolute URLs)
    // Client-side: use relative URL (browser adds origin, reverse proxy handles routing)
    const baseUrl =
      config?.baseUrl ||
      (typeof window === "undefined"
        ? process.env.API_BASE_URL_INTERNAL || "http://localhost:8080/api"
        : process.env.NEXT_PUBLIC_API_BASE_URL || "/api");

    this.config = {
      baseUrl,
      defaultHeaders: {
        ...config?.defaultHeaders,
      },
    };
  }

  async get<T>(endpoint: string, options?: RequestOptions): Promise<T> {
    return this.request<T>(endpoint, {
      method: "GET",
      ...this.applyOptions(undefined, options),
    });
  }

  async post<T>(
    endpoint: string,
    body?: any,
    options?: RequestOptions
  ): Promise<T> {
    return this.request<T>(endpoint, {
      method: "POST",
      ...this.applyOptions(body, options),
    });
  }

  async put<T>(
    endpoint: string,
    body?: any,
    options?: RequestOptions
  ): Promise<T> {
    return this.request<T>(endpoint, {
      method: "PUT",
      ...this.applyOptions(body, options),
    });
  }

  async patch<T>(
    endpoint: string,
    body?: any,
    options?: RequestOptions
  ): Promise<T> {
    return this.request<T>(endpoint, {
      method: "PATCH",
      ...this.applyOptions(body, options),
    });
  }

  async delete<T>(endpoint: string, options?: RequestOptions): Promise<T> {
    return this.request<T>(endpoint, {
      method: "DELETE",
      ...this.applyOptions(undefined, options),
    });
  }

  getBaseUrl(): string {
    return this.config.baseUrl;
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit & { skipAuth?: boolean; params?: Record<string, string | number | undefined> }
  ): Promise<T> {
    const url = `${this.config.baseUrl}${this.buildUrl(endpoint, options.params)}`;
    const requestId = apiLogger.generateRequestId();
    const requestStartTime = Date.now();

    const headers: Record<string, string> = {
      ...this.config.defaultHeaders,
      ...(options.headers as Record<string, string> | undefined),
    };

    // Mock-auth E2E: forward the X-Test-Org-ID identity on calls so the
    // backend mock middleware can resolve the org/account. The backend prefers
    // this header over the Bearer token, and the mock JWT adapter cannot
    // resolve a real org on its own. Works both server-side (next/headers) and
    // client-side (document cookie set by the e2e storage state).
    if (
      process.env.AUTH_MOCK_ENABLED === "true" &&
      !headers["X-Test-Org-ID"]
    ) {
      try {
        let mockOrgId: string | null = null;
        if (typeof window === "undefined") {
          const { cookies } = await import("next/headers");
          mockOrgId = (await cookies()).get("X-Test-Org-ID")?.value ?? null;
        } else {
          mockOrgId =
            document.cookie.match(/(?:^|; )X-Test-Org-ID=([^;]*)/)?.[1] ?? null;
        }
        if (mockOrgId) {
          headers["X-Test-Org-ID"] = mockOrgId;
        }
      } catch {
        // cookies() unavailable (non-request context) — ignore.
      }
    }

    const { skipAuth, ...restOptions } = options;
    const { params: _params, ...fetchOptions } = restOptions;
    const shouldAttachAuth = !skipAuth && !headers["Authorization"];
    let attachedAccessToken: string | null = null;

    if (shouldAttachAuth) {
      const token = await resolveAccessToken();
      if (token) {
        headers["Authorization"] = `Bearer ${token}`;
        attachedAccessToken = token;
      }
    } else {
      const headerValue = headers["Authorization"];
      attachedAccessToken =
        typeof headerValue === "string" && headerValue.startsWith("Bearer ")
          ? headerValue.slice("Bearer ".length)
          : null;
    }

    const requestInit: RequestInit = {
      ...fetchOptions,
      headers,
      credentials: "include",
    };

    // Log outgoing request
    apiLogger.logRequest(requestId, {
      method: options.method || "GET",
      url,
      headers,
      body: options.body,
      timestamp: requestStartTime,
    });

    try {
      const response = await fetch(url, requestInit);
      const responseTime = Date.now();

      // Extract response headers
      const responseHeaders: Record<string, string> = {};
      response.headers.forEach((value, key) => {
        responseHeaders[key] = value;
      });

      // Clone response to read body for logging
      const responseClone = response.clone();
      let responseBody: any;
      try {
        responseBody = await responseClone.json();
      } catch {
        // Response is not JSON
        responseBody = null;
      }

      // Log response
      apiLogger.logResponse(requestId, {
        status: response.status,
        statusText: response.statusText,
        headers: responseHeaders,
        body: responseBody,
        duration: responseTime - requestStartTime,
        timestamp: responseTime,
      });

      if (response.status === 401 && !skipAuth) {
        return this.handleUnauthorizedResponse<T>({
          url,
          requestInit,
          headers,
          attachedAccessToken,
        });
      }

      if (!response.ok) {
        const error = await this.buildApiError(response);

        // Log error response
        apiLogger.logError(requestId, {
          message: error.message,
          status: response.status,
          details: error,
          duration: Date.now() - requestStartTime,
          timestamp: Date.now(),
        });

        throw error;
      }

      return response.json() as Promise<T>;
    } catch (error) {
      // Log network or other errors
      apiLogger.logError(requestId, {
        message: error instanceof Error ? error.message : "Unknown error",
        details: error,
        duration: Date.now() - requestStartTime,
        timestamp: Date.now(),
      });

      throw error;
    }
  }

  private buildUrl(
    endpoint: string,
    params?: Record<string, string | number | undefined>
  ): string {
    if (!params) return endpoint;

    const searchParams = new URLSearchParams();
    for (const [key, value] of Object.entries(params)) {
      if (value !== undefined) {
        searchParams.set(key, String(value));
      }
    }

    const queryString = searchParams.toString();
    if (!queryString) return endpoint;

    return endpoint.includes("?")
      ? `${endpoint}&${queryString}`
      : `${endpoint}?${queryString}`;
  }

  private prepareBody(body: any): BodyInit | undefined {
    if (!body) return undefined;
    if (body instanceof FormData) return body;
    return JSON.stringify(body);
  }

  private applyOptions(
    body: any,
    options?: RequestOptions
  ): RequestInit & { skipAuth?: boolean; params?: Record<string, string | number | undefined> } {
    const headers = this.prepareHeaders(body, options?.headers);

    return {
      body: this.prepareBody(body),
      headers,
      skipAuth: options?.skipAuth,
      params: options?.params,
    };
  }

  private prepareHeaders(
    body: any,
    customHeaders?: Record<string, string>
  ): Record<string, string> {
    const headers = { ...customHeaders };

    if (!(body instanceof FormData) && body && !headers["Content-Type"]) {
      headers["Content-Type"] = "application/json";
    }

    return headers;
  }

  private async handleUnauthorizedResponse<T>({
    url,
    requestInit,
    headers,
    attachedAccessToken,
  }: {
    url: string;
    requestInit: RequestInit;
    headers: Record<string, string>;
    attachedAccessToken: string | null;
  }): Promise<T> {
    const tokenState = classifyTokenState(attachedAccessToken);

    const refreshedToken =
      tokenState === "valid"
        ? await resolveAccessToken({ forceRefresh: true })
        : await refreshToken();

    if (!refreshedToken) {
      await logoutUser();
      throw new Error("Session expired");
    }

    headers["Authorization"] = `Bearer ${refreshedToken}`;

    const retryResponse = await fetch(url, {
      ...requestInit,
      headers,
    });

    if (!retryResponse.ok) {
      if (retryResponse.status === 401) {
        await logoutUser();
        throw new Error("Session expired");
      }

      throw await this.buildApiError(retryResponse);
    }

    return retryResponse.json() as Promise<T>;
  }

  private async buildApiError(response: Response): Promise<Error> {
    let errorMessage = response.statusText;

    try {
      const errorData = await response.json();
      if (errorData && typeof errorData.message === "string") {
        errorMessage = errorData.message;
      }
    } catch {
      // Ignore JSON parsing errors and fall back to status text.
    }

    return new Error(`API Error ${response.status}: ${errorMessage}`);
  }
}

// Export singleton instance
export const apiClient = new ApiClient();

type ResolveAccessTokenOptions = {
  forceRefresh?: boolean;
};

let cachedToken: string | null = null;
let cachedPayload: AccessTokenPayload | null = null;
let stytchClientPromise: Promise<any> | null = null;
let refreshPromise: Promise<string | null> | null = null;
let refreshPromiseTimeout: NodeJS.Timeout | null = null;

// Retry configuration
const MAX_REFRESH_RETRIES = 3;
const REFRESH_RETRY_DELAYS = [1000, 2000, 4000]; // Exponential backoff: 1s, 2s, 4s
const REFRESH_PROMISE_TIMEOUT_MS = 10000; // 10 seconds max wait for shared refresh

export async function resolveAccessToken(
  options: ResolveAccessTokenOptions = {}
): Promise<string | null> {
  const { forceRefresh = false } = options;

  if (!forceRefresh && cachedToken && cachedPayload && !isTokenExpired(cachedPayload)) {
    return cachedToken;
  }

  if (!forceRefresh) {
    const storedToken = await readStoredAccessToken();

    if (storedToken) {
      const payload = decodeAccessToken(storedToken);
      if (payload && !isTokenExpired(payload)) {
        updateCachedToken(storedToken, payload);

        if (typeof window !== "undefined") {
          persistBrowserSessionJwt(storedToken);
        }

        return storedToken;
      }
    }
  }

  const refreshedToken = await refreshToken();
  if (refreshedToken) {
    return refreshedToken;
  }

  await clearSessionCookies();
  return null;
}

// Helper function to wait/sleep for retry delays
async function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

// Helper function to perform refresh with retry logic
async function refreshTokenWithRetry(attempt: number = 0): Promise<string | null> {
  try {
    const token = await performTokenRefresh();

    if (!token) {
      if (attempt < MAX_REFRESH_RETRIES - 1) {
        const delay = REFRESH_RETRY_DELAYS[attempt];
        await sleep(delay);
        return refreshTokenWithRetry(attempt + 1);
      }
      return null;
    }

    const payload = decodeAccessToken(token);
    if (!payload || isTokenExpired(payload)) {
      // Retry if token is invalid and we have retries left
      if (attempt < MAX_REFRESH_RETRIES - 1) {
        const delay = REFRESH_RETRY_DELAYS[attempt];
        resetCachedToken();
        await sleep(delay);
        return refreshTokenWithRetry(attempt + 1);
      }

      resetCachedToken();
      return null;
    }

    updateCachedToken(token, payload);

    if (typeof window !== "undefined") {
      persistBrowserSessionJwt(token);
    }

    return token;
  } catch {
    if (attempt < MAX_REFRESH_RETRIES - 1) {
      const delay = REFRESH_RETRY_DELAYS[attempt];
      await sleep(delay);
      return refreshTokenWithRetry(attempt + 1);
    }

    resetCachedToken();
    return null;
  }
}

export async function refreshToken(): Promise<string | null> {
  if (!refreshPromise) {
    // Set up timeout for the refresh promise
    if (refreshPromiseTimeout) {
      clearTimeout(refreshPromiseTimeout);
    }

    refreshPromise = refreshTokenWithRetry()
      .finally(() => {
        if (refreshPromiseTimeout) {
          clearTimeout(refreshPromiseTimeout);
          refreshPromiseTimeout = null;
        }
        refreshPromise = null;
      });

    // Add timeout to prevent hanging on shared refresh promise
    refreshPromiseTimeout = setTimeout(() => {
      if (refreshPromise) {
        refreshPromise = null;
      }
      refreshPromiseTimeout = null;
    }, REFRESH_PROMISE_TIMEOUT_MS);
  }

  return refreshPromise;
}

async function performTokenRefresh(): Promise<string | null> {
  if (typeof window === "undefined") {
    return tryRefreshServerToken();
  }

  return fetchSessionJwtFromApi();
}

async function readStoredAccessToken(): Promise<string | null> {
  if (typeof window === "undefined") {
    try {
      const { cookies } = await import("next/headers");
      const cookieStore = await cookies();
      return cookieStore.get(SESSION_JWT_COOKIE_NAME)?.value ?? null;
    } catch {
      return null;
    }
  }

  return readBrowserCookie(SESSION_JWT_COOKIE_NAME);
}

async function readServerSessionToken(): Promise<string | null> {
  if (typeof window !== "undefined") {
    return null;
  }

  try {
    const { cookies } = await import("next/headers");
    const cookieStore = await cookies();
    return cookieStore.get(SESSION_COOKIE_NAME)?.value ?? null;
  } catch {
    return null;
  }
}

async function clearSessionCookies(): Promise<void> {
  resetCachedToken();
  const expiry = "Thu, 01 Jan 1970 00:00:00 GMT";

  if (typeof document !== "undefined") {
    deleteBrowserCookie(SESSION_COOKIE_NAME, expiry);
    deleteBrowserCookie(SESSION_JWT_COOKIE_NAME, expiry);
    return;
  }

  try {
    const nextHeaders = await import("next/headers");
    const cookieStore = await nextHeaders.cookies();
    const mutableStore = cookieStore as unknown as {
      delete?: (name: string) => void;
    };

    mutableStore.delete?.(SESSION_COOKIE_NAME);
    mutableStore.delete?.(SESSION_JWT_COOKIE_NAME);
  } catch {
    // Swallow errors - this best-effort cleanup should not break callers.
  }
}

async function logoutUser(): Promise<void> {
  await clearSessionCookies();

  if (typeof window !== "undefined") {
    const currentPath = window.location.pathname + window.location.search;
    const returnTo = encodeURIComponent(currentPath);
    const target = AUTH_ROUTES.LOGIN ?? "/login";
    window.location.href = `${target}?returnTo=${returnTo}`;
  }
}

function deleteBrowserCookie(name: string, expiry: string) {
  document.cookie = `${name}=; path=/; expires=${expiry}; max-age=0`;
}

function readBrowserCookie(name: string): string | null {
  if (typeof document === "undefined") {
    return null;
  }

  const escapedName = escapeRegExp(name);
  const match = document.cookie.match(
    new RegExp(`(?:^|; )${escapedName}=([^;]*)`)
  );
  return match ? decodeURIComponent(match[1]) : null;
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function updateCachedToken(
  token: string | null,
  payload: AccessTokenPayload | null
) {
  cachedToken = token;
  cachedPayload = payload;
}

function resetCachedToken() {
  cachedToken = null;
  cachedPayload = null;
}

// Export for logout cleanup
export { resetCachedToken };

async function exchangeSessionTokenForJwt(
  sessionToken: string,
  sessionDurationMinutes: number = 480 // Default 8 hours
): Promise<string | null> {
  if (typeof window !== "undefined") {
    return null;
  }

  try {
    const client = await loadStytchB2BClient();
    if (!client) {
      return null;
    }

    const response = (await client.sessions.authenticate({
      session_token: sessionToken,
      session_duration_minutes: sessionDurationMinutes,
    } as any)) as { session_jwt?: string | null };

    return response?.session_jwt ?? null;
  } catch {
    return null;
  }
}

async function loadStytchB2BClient(): Promise<any | null> {
  if (typeof window !== "undefined") {
    return null;
  }

  if (!stytchClientPromise) {
    stytchClientPromise = createStytchB2BClient().catch(() => {
      stytchClientPromise = null;
      return null;
    });
  }

  return stytchClientPromise;
}

async function createStytchB2BClient(): Promise<any | null> {
  // Ensure server-only execution
  if (typeof window !== "undefined") {
    throw new Error("createStytchB2BClient must only be called on server");
  }

  const projectId = process.env.STYTCH_PROJECT_ID;
  const secret = process.env.STYTCH_SECRET;

  if (!projectId || !secret) {
    return null;
  }

  const projectEnv =
    process.env.STYTCH_PROJECT_ENV ||
    process.env.NEXT_PUBLIC_STYTCH_PROJECT_ENV ||
    "test";

  const { default: Stytch, envs } = await import("stytch");

  return new Stytch.B2BClient({
    project_id: projectId,
    secret,
    env: projectEnv === "live" ? envs.live : envs.test,
  });
}

async function tryRefreshServerToken(): Promise<string | null> {
  const sessionToken = await readServerSessionToken();
  if (!sessionToken) {
    return null;
  }

  // Use default duration (480 minutes / 8 hours)
  return exchangeSessionTokenForJwt(sessionToken);
}

function persistBrowserSessionJwt(token: string): void {
  if (typeof window === "undefined") {
    return;
  }

  // Use hardcoded default for browser context (8 hours)
  const maxAgeSeconds = 480 * 60; // 28800 seconds
  const secure = window.location.protocol === "https:";

  const cookieParts = [
    `${SESSION_JWT_COOKIE_NAME}=${encodeURIComponent(token)}`,
    "path=/",
    `max-age=${maxAgeSeconds}`,
    "SameSite=Lax",
  ];

  if (secure) {
    cookieParts.push("Secure");
  }

  document.cookie = cookieParts.join("; ");
}

async function fetchSessionJwtFromApi(): Promise<string | null> {
  try {
    const response = await fetch("/api/auth/session/refresh", {
      method: "POST",
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
      },
    });

    if (!response.ok) {
      return null;
    }

    const data = await response.json();
    const token =
      typeof data?.sessionJwt === "string" && data.sessionJwt.length > 0
        ? data.sessionJwt
        : null;

    if (token) {
      persistBrowserSessionJwt(token);
    }

    return token;
  } catch {
    return null;
  }
}

type TokenState = "none" | "invalid" | "expired" | "valid";

function classifyTokenState(token: string | null): TokenState {
  if (!token) {
    return "none";
  }

  const payload = decodeAccessToken(token);
  if (!payload) {
    return "invalid";
  }

  return isTokenExpired(payload) ? "expired" : "valid";
}
