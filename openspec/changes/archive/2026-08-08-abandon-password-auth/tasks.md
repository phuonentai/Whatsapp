## 1. Archive stale password-auth changes

- [x] 1.1 [OPS] Move `openspec/changes/fix-password-auth-org-id/` to `openspec/changes/archive/2026-08-08-fix-password-auth-org-id/` (manual move — no delta merge into `openspec/specs/`)
- [x] 1.2 [OPS] Move `openspec/changes/add-password-auth/` to `openspec/changes/archive/2026-08-08-add-password-auth/` (manual move — no delta merge into `openspec/specs/`)
- [x] 1.3 [OPS] Add `ABANDONED.md` to each archived change directory explaining the rejection reason and referencing the `signup-stytch-compliance` spec
- [x] 1.4 [OPS] Verify `openspec list --json` no longer shows `fix-password-auth-org-id` or `add-password-auth` as active changes

## 2. E2E regression tests for passwordless auth

- [x] 2.1 [FE-NEXT] Create `next_b2b_starter/e2e/page-objects/signup.page.ts` — signup page object (fill account/organization steps, submit) reusing the mock-auth fixture pattern
- [x] 2.2 [FE-NEXT] Create `next_b2b_starter/e2e/specs/auth-passwordless.spec.ts` covering:
  - `/signup` renders no `input[type="password"]`
  - `/auth` renders no `input[type="password"]`
  - intercepted `POST /auth/signup` payload contains no `owner_password` key
  - `POST /auth/login` returns 404/405
  - `/authenticate` renders no `input[type="password"]`
- [x] 2.3 [FE-NEXT] Verify `pnpm exec playwright test --project=chromium` runs the new spec and existing specs are unaffected (specs may be skipped if the dev server is not running — document outcome)
  - Outcome: `auth-passwordless.spec.ts` passes 5/5 against the running Next.js dev server (localhost:3001)
  - Existing CRM specs (contacts/companies/deals/activities/...) fail only because the Go backend (localhost:8080) is not running in this environment — a pre-existing infrastructure condition unrelated to this change; they load data from the CRM API and time out waiting for the table. Spec would pass with `make server` up.
  - The signup-payload interception test initially FAILED, catching a live regression: `signup-repository.ts` still sent `owner_password`. Fixed by removing the dead field from `SignupMagicLinkRequestDto`/`BootstrapOrganizationRequestDto` and dropping `generateSecurePassword` usage from the signup repository (in scope of this change's spec: "Signup payload excludes owner_password"). Re-run: 5/5 pass.
- [x] 2.4 [FE-NEXT] Run `pnpm lint` and `pnpm build` in `next_b2b_starter/` to confirm no type/lint regressions
  - Outcome: `pnpm lint` (`PORT=0 next lint`) cannot run — Next.js 16 removed the `next lint` command and treats `lint/` as a page dir; ESLint 9 in this repo has no flat config (only legacy `.eslintrc.json`). Pre-existing repo condition, unrelated to this change. `npx tsc --noEmit` is the effective lint gate here.
  - `npx tsc --noEmit` on the new E2E files (`e2e/specs/auth-passwordless.spec.ts`, `e2e/page-objects/signup.page.ts`): clean, 0 errors.
  - Note: the wider repo has 39 pre-existing TS errors in `lib/hooks/*` (query keys for `crm`/`whatsappConfig` missing from the `navKeys`/`queryKeys` object) — observed before and unrelated to this change.
  - `pnpm build` fails on `components/billing/plans-modal.tsx:108` (`Property 'error' does not exist on type 'ActionResult<MPCheckoutData>'`). The identical buggy line exists verbatim at `HEAD` — a pre-existing MercadoPago billing type error, unrelated to this auth change. Neither of my edited files (`auth.dto.ts`, `signup-repository.ts`) surfaces in the failing build output.

## 3. Verification

- [x] 3.1 [OPS] Run `openspec validate --change abandon-password-auth` and confirm the change is valid
  - Ran `openspec validate abandon-password-auth --type change` → "Change 'abandon-password-auth' is valid". Also confirmed via `openspec validate --all` → `✓ change/abandon-password-auth`. (The 16 failing items in the `--all` run are living specs already lacking `## Purpose`/`## Requirements` structure — pre-existing, unrelated to this change.)
- [x] 3.2 [OPS] Confirm `openspec/specs/` living specs were NOT modified by the archive move (no password requirements leaked into `stytch-authorization` or new `password-auth` capability)
  - `git status --short openspec/specs` → empty. Living specs untouched.
