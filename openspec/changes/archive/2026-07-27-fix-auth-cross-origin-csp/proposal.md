## Why

The auth page makes a direct cross-origin `fetch` to `http://localhost:8080/api/auth/check-email` which is blocked by the Content Security Policy `connect-src` directive. The backend is already reachable via a Next.js rewrite (`/api/:path*` → `localhost:8080/api/:path*`), so the call should use a same-origin relative URL instead.

## What Changes

- Change the default API base URL in the auth page from `http://localhost:8080/api` to `/api`, matching the pattern already used by `ApiClient` (`lib/api/api/client/api-client.ts:35`)

## Capabilities

### New Capabilities

None — this is a bugfix, not a new capability.

### Modified Capabilities

None — no spec-level behavior is changing. The email check flow remains identical; only the URL routing mechanism changes.

## Impact

- **Code**: 1 line changed in `next_b2b_starter/app/auth/page.tsx` (line 139)
- **No API changes**, no dependency changes, no breaking changes
- **Testing**: Verify the auth page login flow successfully checks email and sends magic link
