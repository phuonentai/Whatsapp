## Why

`pnpm lint` in `next_b2b_starter/` is broken: Next 16 removed the `next lint` command, ESLint 9.39 only accepts flat config, and the legacy `.eslintrc.json` (extends `next/core-web-vitals`) crashes under eslintrc mode with a circular-structure error. Every change since the Next 16 upgrade records lint verification as "blocked by pre-existing tooling" (see archived change `2026-08-08-fix-frontend-build` task 5.2), which stalls the governance verification gate for all future frontend work.

## What Changes

- Delete `next_b2b_starter/.eslintrc.json` (legacy, incompatible with ESLint 9).
- Add `next_b2b_starter/eslint.config.mjs` — flat config using `defineConfig`/`globalIgnores` from `eslint/config`, spreading the official `eslint-config-next/core-web-vitals` flat entry (verified present in `eslint-config-next@16.0.10`), with ignores for build and test artifacts (`.next/`, `out/`, `build/`, `test-results/`).
- Add `typescript-eslint@^8.49.0` as a devDependency (the `core-web-vitals` flat entry requires it transitively; it is already in the pnpm store at 8.49.0 and only needs to be declared for pnpm strict resolution).
- Replace the dead `"lint": "PORT=0 next lint"` script in `package.json` with `"lint": "eslint ."`.
- Verify `pnpm lint` exits zero (triaging first-run violations without disabling rules wholesale), `pnpm build` stays green, and `npx tsc --noEmit` stays green.

## Capabilities

### New Capabilities

- none (pure tooling repair; no behavioral requirements introduced)

### Modified Capabilities

- `governance-workflow`: new requirement — verification commands recorded in `tasks.md` SHALL be runnable with the current toolchain; frontend lint verification SHALL use the Next-16-compatible invocation (`eslint .` with flat config); changes SHALL NOT defer verification as "blocked by pre-existing tooling" without an owning tooling-restoration change

## Impact

- **FE**: `next_b2b_starter/package.json`, `next_b2b_starter/pnpm-lock.yaml`, `next_b2b_starter/.eslintrc.json` (deleted), `next_b2b_starter/eslint.config.mjs` (new). No application code, routes, or behavior change.
- **BE**: none. **DB**: none. **Auth boundary**: no change — no Stytch contracts, credentials, sessions, or RBAC touched.
- **Commit sequencing**: `package.json` and `pnpm-lock.yaml` are already staged in a pending CRM commit. This change SHALL be committed separately after the pending commit lands to keep the migrations distinct.
- **Rollback**: restore `.eslintrc.json` from git history and revert the `lint` script; no Stytch state involved (never mutated).

## Non-Goals

- Adding the `eslint-config-next/typescript` flat entry (typescript-eslint `recommended` rules) — deferred; the legacy config never used it.
- Introducing new lint rules, plugins (including Playwright), or CI wiring.
- Wholesale fixing of pre-existing violations beyond triage of the first-run output.
- Local credential storage, password, MFA, or session handling (unchanged; forbidden by constitution).
