## 1. Upgrade Stytch Go SDK

- [x] 1.1 Update `go.mod` to replace `stytch-go/v16` with `stytch-go/v18`
- [x] 1.2 Run `go mod tidy` to resolve transitive dependencies
- [x] 1.3 Fix import paths in `stytch_member_repository.go` for v18 package structure
- [x] 1.4 Fix import paths in `stytch_organization_repository.go` for v18 package structure
- [x] 1.5 Fix import paths in `stytch/client.go` for v18 package structure
- [x] 1.6 Run `go build ./...` to verify compilation

## 2. Restructure Bootstrap Flow

- [x] 2.1 Change `CreateMember` call to set `SendInvite: true` in `member_service_impl.go`
- [x] 2.2 Remove the separate `LoginOrSignup` call (Step 6) from `BootstrapOrganizationWithOwner`
- [x] 2.3 Move `shouldRollback = false` to after the `CreateMember` step (Step 3) succeeds
- [x] 2.4 Ensure `MagicLinkSent` is `true` on successful member creation, `false` on failure
- [x] 2.5 Verify the rollback stack cleans up correctly when `CreateMember` fails

## 3. Structured Error Diagnostics

- [x] 3.1 Define error code constants in `internal/modules/organizations/domain/errors.go`
- [x] 3.2 Map Stytch API errors to error codes in `member_handler.go`
- [x] 3.3 Update `response.Error()` calls to include `code` and `detail` fields
- [x] 3.4 Verify frontend error handling parses structured error responses

## 4. Remove Dead `owner_password` Field

- [x] 4.1 Remove `owner_password` from `SignupMagicLinkRequestDto` in `auth.dto.ts`
- [x] 4.2 Remove password generation from `signup-repository.ts` `createOrganizationWithMagicLink`
- [x] 4.3 Remove `password-generator.ts` if no other consumers exist
- [x] 4.4 Verify the signup form submits and the backend binds the request without the field

## 5. Audit and Cleanup

- [x] 5.1 Search for all callers of `AuthMemberRepository.SendMagicLink` — if only the bootstrap flow called it, remove the method from the interface and implementation
- [x] 5.2 Search for all callers of `MagicLinks.Email.LoginOrSignup` in the go-b2b-starter codebase
- [x] 5.3 Remove unused `SendMagicLink` domain types and repository methods if no other callers exist
- [x] 5.4 Run full backend test suite to verify no regressions

## 6. Verification

- [x] 6.1 Start the backend and verify it initializes Stytch client successfully
- [ ] 6.2 Test signup end-to-end: fill form, submit, verify invite email received
- [ ] 6.3 Click magic link from invite email and verify redirect to `/authenticate` completes login
- [ ] 6.4 Verify the 500 error is no longer returned on valid signup

> **Note:** Stytch client initializes successfully (no placeholder warning in logs). Full server startup fails on a pre-existing paywall module DI issue (`missing type: adapters.OrganizationStore`) — unrelated to this change. The old server on port 8080 uses the previous code. Once the paywall DI issue is resolved, restart with:

```bash
cd go-b2b-starter && go run ./cmd/api/main.go
