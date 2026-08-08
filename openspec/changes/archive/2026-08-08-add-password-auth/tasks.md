## 1. Backend: Add password to bootstrap request DTO and service

- [ ] 1.1 [BE-DOMAIN] Add `OwnerPassword` field to `BootstrapOrganizationRequest` struct with JSON tag `owner_password` and `required,min=8` binding
- [ ] 1.2 [BE-DOMAIN] Update `Validate()` method to check password length (min 8 chars)
- [ ] 1.3 [BE-DOMAIN] Pass password through the bootstrap flow: `member_handler.go` → `member_service.go` → `member_service_impl.go`

## 2. Backend: Set password via Stytch REST API

- [ ] 2.1 [BE-INFRA] Add `CreateMemberPassword` method to `AuthMemberRepository` interface in `domain/auth_provider.go`
- [ ] 2.2 [BE-INFRA] Implement `CreateMemberPassword` in `stytch_member_repository.go` — raw HTTP POST to `https://test.stytch.com/v1/b2b/passwords` with `organization_id`, `member_id`, `password`
- [ ] 2.3 [BE-INFRA] In `member_service_impl.go`, replace `InviteMember` call with `CreateMember` (no invite) + `CreateMemberPassword` call in the bootstrap flow
- [ ] 2.4 [BE-INFRA] Update response to set `MagicLinkSent: false` since no invite email is sent

## 3. Backend: Add login endpoint

- [ ] 3.1 [BE-DOMAIN] Add login request/response types to `member_service.go`
- [ ] 3.2 [BE-DOMAIN] Add `Authenticate` method to `AuthMemberRepository` interface
- [ ] 3.3 [BE-INFRA] Implement `Authenticate` in `stytch_member_repository.go` using `r.client.API().Passwords.Authenticate()`
- [ ] 3.4 [BE-INFRA] Add `Login` handler in `member_handler.go` — accepts email + password, calls service layer
- [ ] 3.5 [BE-INFRA] Add `POST /api/auth/login` route in routes file
- [ ] 3.6 [BE-INFRA] Wire login service + handler dependencies in DI

## 4. Frontend: Update signup form

- [ ] 4.1 [FE-NEXT] Add `owner_password` field to `SignupMagicLinkRequestDto` in `auth.dto.ts`
- [ ] 4.2 [FE-NEXT] Add password input (type=password, minLength=8) to the signup form in `app/signup/page.tsx`
- [ ] 4.3 [FE-NEXT] Update `signup-repository.ts` to send `owner_password` in the payload

## 5. Frontend: Update login page

- [ ] 5.1 [FE-NEXT] Update login form in `app/auth/page.tsx` to include password input alongside email
- [ ] 5.2 [FE-NEXT] Add login server action or API call that sends `POST /api/auth/login` with email + password
- [ ] 5.3 [FE-NEXT] Keep magic link option as a toggle, add a "or sign in with password" divider

## 6. Verify

- [ ] 6.1 [BE-INFRA] Run `go build ./cmd/api/main.go` to verify compilation
- [ ] 6.2 [FE-NEXT] Run `pnpm build` to verify frontend compilation
- [ ] 6.3 [BE-INFRA] Rebuild Docker images, restart, and test signup with password end-to-end
- [ ] 6.4 [BE-INFRA] Test login with email + password
