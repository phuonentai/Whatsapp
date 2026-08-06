## Context

The password login flow (`POST /auth/login`) calls Stytch's `Passwords.Authenticate` with an empty `OrganizationID`. The Stytch B2B API requires the organization ID to locate the member — without it, Stytch returns `organization_not_found`.

The `Login` service method in `member_service_impl.go` currently passes through directly to the repository's `Authenticate`, which hardcodes `OrganizationID: ""`. The service has access to `localAccountRepo` and the existing org mapping tables — it can resolve the Stytch org ID from the member's email before authenticating.

```
CURRENT FLOW:
  POST /auth/login { email, password }
    → Login() service
      → repo.Authenticate({ Email, Password, OrganizationID: "" })
        → Stytch Passwords.Authenticate(OrganizationID:"")
          → 404 organization_not_found ✗

REQUIRED FLOW:
  POST /auth/login { email, password }
    → Login() service
      → Lookup account by email → get local_org_id
      → Lookup Stytch org_id from local_org_id
      → repo.Authenticate({ Email, Password, OrganizationID: resolved_id })
        → Stytch Passwords.Authenticate(OrganizationID: resolved_id)
          → 200 session_token + session_jwt ✓
```

## Goals / Non-Goals

**Goals:**
- Resolve the member's Stytch organization ID from the local database before calling Stytch `Passwords.Authenticate`
- Fix the 500/401 error on password login

**Non-Goals:**
- Changing the signup flow or password migration
- Adding password reset or "forgot password" endpoints
- Modifying the frontend login page or auth UI
- Adding Cross-Organization password support

## Decisions

### Decision 1: Resolve org via email lookup

The `Login` method will:
1. Query `localAccountRepo` to find the account by email
2. From the account's `local_org_id`, look up the Stytch organization ID via `localOrgRepo` or the existing org mapping
3. Pass the resolved Stytch org ID into the `Authenticate` call

**Alternative considered:** Let Stytch resolve the org (using Cross-Organization passwords). Rejected because Cross-Organization mode requires a different API endpoint pattern and more complex session handling. Organization-scoped passwords are the current configuration.

**Alternative considered:** Accept `org_id` from the frontend login request. Rejected because it pushes routing logic to the client and requires the frontend to know the org ID, which it doesn't have before login.

### Decision 2: Add OrganizationID to LoginRequest domain type

Add an `OrganizationID string` field to `domain.LoginRequest` so the resolved ID flows through the domain boundary cleanly.

## Risks / Trade-offs

| Risk | Mitigation |
|------|-----------|
| Email resolves to multiple orgs (Cross-Organization) | Not currently configured. If enabled later, will need a different flow. |
| Account not found in local DB (race on signup) | Returns clear error. The signup flow creates the local account synchronously. |
| Stytch org ID not mapped locally | Returns `INVALID_CREDENTIALS` — same behavior as incorrect password. |