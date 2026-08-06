## Context

The current architecture uses a Go backend (go-b2b-starter/) for session management, JWT validation, and auth middleware, with the Next.js frontend (next_b2b_starter/) making synchronous backend calls for every protected route. This creates unnecessary latency and custom code for what Stytch provides out of the box via its pre-built frontend SDKs. The dual SSOT architecture (go-b2b-starter as static SSOT, Stytch B2B as runtime SSOT) is established but is not fully leveraged on the frontend side.

## Goals / Non-Goals

**Goals:**
- Eliminate custom auth page code (login, signup, settings) entirely — use pre-built Stytch components
- Move session validation from the Go backend to the Next.js edge layer for latency-free protected route gating
- Reduce total authentication-related code to under 100 lines across both frontend and backend
- Enforce `organization_id`-scoped queries from the verified JWT claim at every data access point
- Maintain backward compatibility with existing Go backend Stytch v18 SDK integration

**Non-Goals:**
- Modifying the Go backend's Stytch v18 SDK integration or its RBAC permission resolution logic
- Creating custom session tables or replicating Stytch's session store locally
- Replacing Stytch as the identity provider or RBAC authority
- Rewriting existing CRM/Spanish-language UI beyond replacing auth/settings pages

## Decisions

### Decision 1: Edge Middleware over Backend Session Proxy

- **Choice**: Validate `stytch_session_jwt` cookies at the Next.js edge via `@stytch/nextjs/b2b/edge` `createClient().sessions.authenticateJwt()`
- **Alternatives considered**: (a) Proxy all requests through Go backend middleware for JWT validation — rejected because it adds latency and defeats the purpose of edge caching; (b) Client-side-only checks — rejected because it's insecure for protected routes
- **Rationale**: Stytch's edge SDK validates the JWT signature statelessly using locally cached JWKS public keys, requiring zero database calls or backend round-trips. This preserves the "zero custom session infrastructure" invariant.

### Decision 2: Signal Via X-Forwarded-Auth Instead of Removing Go Validation

- **Choice**: The edge middleware sets `X-Forwarded-Auth: true` and the JWT claims as request headers before forwarding to the Go API; Go backend MAY use this signal to skip redundant Stytch token introspection
- **Alternatives considered**: (a) Completely remove Go middleware session checks — risky during rollout; (b) Only do edge validation — breaks API calls not routed through Next.js
- **Rationale**: Defense-in-depth — the Go backend can independently verify if desired, but can also trust the edge signal for performance. This allows phased rollout and easy rollback.

### Decision 3: AuthFlowType.Discovery for Login

- **Choice**: Use `AuthFlowType.Discovery` in the Stytch B2B config so users can discover and join/select their organization on login
- **Rationale**: Standard Stytch B2B pattern for multi-tenant apps. Avoids building custom org selection UI.

### Decision 4: Shared Schema with organization_id via JWT Claim

- **Choice**: Pass `organization_id` from JWT claim through middleware headers → API gateway → repository layer; enforce in queries via param binding rather than relying solely on RLS
- **Alternatives considered**: (a) RLS-only isolation — complex to debug, harder to test; (b) Separate databases per tenant — overkill for this project
- **Rationale**: JWT claim is the authoritative source; query-level filtering provides explicit, testable isolation. RLS can be added later as defense-in-depth.

### Decision 5: JWKS Fetch Circuit Breaker at Edge

- **Choice**: Wrap the JWKS fetch in a circuit breaker pattern at the edge layer
- **Alternatives considered**: (a) Let the Stytch SDK handle retries — no visibility into failure rates; (b) Fail open on fetch failure — security risk
- **Rationale**: If JWKS fetch fails, the edge cannot validate new tokens, so it must fail secure (redirect to 500). Existing sessions remain valid as long as the cached key works. This mirrors the Go backend circuit breaker protocol.

## Risks / Trade-offs

- **[Risk] Stale JWKS cache allows revoked sessions to pass temporarily** → Mitigation: Set JWKS cache TTL ≤ 300s; high-privilege routes (e.g., billing, org deletion) can trigger an explicit Stytch API introspection
- **[Risk] Split-brain between edge and Go backend validation** → Mitigation: `X-Forwarded-Auth` header signals pre-validation; Go backend logs mismatches for monitoring
- **[Risk] Pre-built components limit UI customization** → Trade-off: Acceptable trade for zero maintenance on auth UI. Stytch provides theming (CSS variables) if customization is needed later
- **[Risk] Rollback requires migrating auth back to custom pages** → Mitigation: Keep old auth page components in repository (unused); `git revert` restores them
