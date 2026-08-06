## 1. Domain Types

- [x] 1.1 Add `OrganizationID string` field to `LoginRequest` in `domain/auth_provider.go`
- [x] 1.2 Update `LoginRequest.Validate()` if applicable

## 2. Service Layer

- [x] 2.1 Update `member_service_impl.go` `Login()` method:
  - Look up account by email via `localAccountRepo.GetByEmail()`
  - Resolve local org → Stytch org ID via `localOrgRepo` or org mapping
  - Populate `OrganizationID` in `LoginRequest` before calling `Authenticate`
- [x] 2.2 Handle error cases: email not found (return `INVALID_CREDENTIALS`), org mapping missing (return `INVALID_CREDENTIALS`)

## 3. Infrastructure Layer

- [x] 3.1 Update `stytch_member_repository.go` `Authenticate()` to use the `OrganizationID` from `LoginRequest` instead of empty string

## 4. Verification

- [ ] 4.1 Start backend and verify `go build ./...` compiles
- [ ] 4.2 Test signup with password, then login with email + password via `POST /auth/login`
- [ ] 4.3 Verify login returns session token and session JWT (not `organization_not_found`)