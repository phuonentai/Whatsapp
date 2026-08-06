## Context

The auth page (`app/auth/page.tsx`) checks whether a user's email exists before sending a magic link. It calls `GET /api/auth/check-email` on the Go backend. The frontend has two paths to reach this endpoint:

1. **Direct cross-origin**: `fetch("http://localhost:8080/api/auth/check-email")` — hits the Go backend on port 8080 directly. This violates CSP `connect-src` which only allows `'self'` (port 3000).
2. **Same-origin via rewrite**: `fetch("/api/auth/check-email")` — goes through Next.js which proxies to Go via the existing rewrite (`/api/:path*` → `http://localhost:8080/api/:path*`).

The auth page currently defaults to path 1 when `NEXT_PUBLIC_API_BASE_URL` is unset. The proper `ApiClient` (`api-client.ts:35`) already uses path 2 on the client side.

## Goals / Non-Goals

**Goals:**
- Make the auth page's default API URL same-origin (`/api`) so CSP doesn't block the email check call
- Align the auth page with the pattern already used by `ApiClient`

**Non-Goals:**
- Changing any other fetch calls (only this one is affected)
- Modifying the CSP header configuration
- Changing the Go backend or the Next.js rewrite

## Decisions

**Decision: Change the default fallback from `http://localhost:8080/api` to `/api`**

- **Rationale**: `ApiClient` already uses `"/api"` as the client-side default (line 35). The auth page is the outlier. Using `/api` routes through the existing Next.js rewrite, avoiding CSP entirely. This also works in every environment without per-environment CSP configuration.
- **Alternatives considered**:
  - Add `localhost:8080` to CSP — fixes the symptom but doesn't address the architectural inconsistency. Also environment-specific.
  - Switch to `ApiClient` — would work, but overkill for one fetch call. The raw fetch is acceptable; only the URL needs fixing.

## Risks / Trade-offs

- **Risk**: In a deployment where `NEXT_PUBLIC_API_BASE_URL` is intentionally set to an absolute URL (e.g., Docker with separate API host), this change has no effect — the env var takes precedence over the default.
- **Risk**: If the Next.js rewrite were removed or misconfigured, the relative `/api` path would fail. Mitigation: The rewrite is a core piece of infrastructure used by all other API calls via `ApiClient`; if it breaks, everything else is already broken.
