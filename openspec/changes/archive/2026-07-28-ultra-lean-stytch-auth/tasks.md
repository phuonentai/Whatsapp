## 1. Frontend Dependencies & Configuration

- [x] 1.1 [FE-NEXT] Add `@stytch/nextjs` and `@stytch/vanilla-js` packages to `next_b2b_starter/package.json`
- [x] 1.2 [FE-NEXT] Add Stytch environment variables (`NEXT_PUBLIC_STYTCH_PUBLIC_TOKEN`, `STYTCH_SECRET`, `STYTCH_PROJECT_ID`) to `.env.local` and validate at startup
- [x] 1.3 [FE-NEXT] Create Stytch client provider wrapper in `lib/stytch.ts` for the frontend (not edge) with B2B configuration

## 2. Edge Middleware Session Validation

- [x] 2.1 [FE-NEXT] Create `proxy.ts` in `next_b2b_starter/` with JWT session validation
- [x] 2.2 [FE-NEXT] Implement JWT cookie extraction and stateless validation using `jwt-decode`
- [x] 2.3 [FE-NEXT] Add protected route matcher config targeting `/dashboard/:path*`, `/settings/:path*`, `/api/protected/:path*`
- [x] 2.4 [FE-NEXT] Implement redirect-to-login flow for missing/invalid/expired JWTs with cookie clearing
- [x] 2.5 [FE-NEXT] Add `X-Forwarded-Auth: true`, `X-Stytch-Organization-Id`, and `X-Stytch-Member-Id` headers on validated requests
- [x] 2.6 [FE-NEXT] Write integration tests covering valid JWT, expired JWT, missing JWT, and malformed JWT scenarios
- [ ] 2.7 [FE-NEXT] Verify JWKS cache behavior: confirm no outgoing HTTP call on subsequent requests after initial fetch

## 3. Replace Login Page with Stytch B2B Component

- [x] 3.1 [FE-NEXT] Rewrite `app/auth/page.tsx` to render `<StytchB2B />` with `AuthFlowType.Discovery`, `B2BProducts.emailMagicLinks`, and `B2BProducts.sso`
- [x] 3.2 [FE-NEXT] Remove all custom form components, password fields, and session generation code from login page
- [ ] 3.3 [FE-NEXT] Verify login flow end-to-end: user enters email → magic link sent → redirect to authenticated session

## 4. Replace Settings Page with Stytch Admin Portal

- [x] 4.1 [FE-NEXT] Rewrite `app/dashboard/settings/components/settings-content.tsx` to render `<AdminPortalMemberManagement />` and `<AdminPortalSSO />`
- [x] 4.2 [FE-NEXT] Remove all custom member management and SSO form components and utilities
- [ ] 4.3 [FE-NEXT] Verify settings page renders admin components and invite/SSO flows work

## 5. Go Backend Edge-Aware Middleware

- [x] 5.1 [BE-INFRA] Update Go session middleware to check for `X-Forwarded-Auth: true` header before attempting Stytch API token introspection
- [x] 5.2 [BE-INFRA] Add validation of `X-Stytch-Organization-Id` and `X-Stytch-Member-Id` header UUID formats in middleware
- [x] 5.3 [BE-INFRA] Add warning-level logging for mismatches between edge auth headers and independent Stytch API validation results
- [x] 5.4 [BE-INFRA] Write integration tests: request with valid edge headers passes through, request with malformed headers returns 401
- [ ] 5.5 [BE-INFRA] Verify mock Stytch API fallback: circuit breaker behavior when Stytch API is unreachable

## 6. Tenant Data Isolation

- [x] 6.1 [BE-DOMAIN] Add middleware that extracts `X-Stytch-Organization-Id` from request context and sets it on the application request scope
- [x] 6.2 [BE-DOMAIN] Audit all tenant-scoped repository queries and add `WHERE organization_id = $1` parameter binding where missing
- [ ] 6.3 [DB-SQLC] Write integration tests confirming cross-tenant access returns empty results (not 401)
- [x] 6.4 [DB-SQLC] (Optional) Add PostgreSQL RLS policies on tenant-scoped tables with `app.current_organization_id` session variable

## 7. Verification & Audit

- [x] 7.1 [FE-NEXT] Run `cloc` or equivalent to confirm authentication-related code is under 100 lines across `app/` and `proxy.ts`
- [x] 7.2 [FE-NEXT] Verify no custom `<form>` elements, password fields, or session generation code exist in auth pages
- [ ] 7.3 [BE-INFRA] Confirm existing test suite still passes: `make test` in `go-b2b-starter/`
- [x] 7.4 [FE-NEXT] Confirm frontend builds without errors: `pnpm build` in `next_b2b_starter/`
