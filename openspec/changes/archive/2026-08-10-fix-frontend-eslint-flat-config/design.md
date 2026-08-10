## Context

`next_b2b_starter/` runs Next.js 16.0.10 with ESLint 9.39.1. Three stacked breaks make `pnpm lint` unusable:

1. Next 16 removed the `next lint` subcommand (only the `next` binary remains in `node_modules/next/dist/bin/`), yet the `lint` script still calls `PORT=0 next lint`.
2. ESLint 9.39 is flat-config-only; the repo's legacy `.eslintrc.json` (`extends: ["next/core-web-vitals"]`) is rejected, and forced eslintrc mode crashes with a circular-structure error because `eslint-config-next@16.0.10` no longer emits a compatible legacy bundle.
3. The official flat entry `eslint-config-next/core-web-vitals` (and its base `index`) `require("typescript-eslint")`, which is present in the pnpm store (8.49.0) but NOT declared as a direct devDependency — pnpm's strict node_modules makes it unresolvable at the package top level.

Every frontend change since the upgrade records lint as "blocked by pre-existing tooling" (e.g., archived `2026-08-08-fix-frontend-build`, task 5.2), which blocks the OpenSpec verification gate for lint-verified tasks.

## Goals / Non-Goals

**Goals:**
- Restore a runnable, Next-16-compatible lint command: `pnpm lint` → `eslint .` exits zero.
- Behavior parity with the legacy config: lint the same code surface with the same Next/React rule set (`core-web-vitals`), no new rules, no type-aware linting.
- Pin the one missing transitive dependency explicitly (`typescript-eslint`) so pnpm resolution is deterministic.
- Keep the change commit-separated from the pending staged CRM commit.

**Non-Goals:**
- Adopting the `eslint-config-next/typescript` flat entry (typescript-eslint `recommended`) — that is a *new* rule surface, deliberately deferred.
- Configuring CI, pre-commit hooks, or the Playwright ESLint plugin.
- Rewriting lint-discovered violations beyond triage (fix trivial ones, record the rest).
- Any backend, database, or auth-boundary change.

## Decisions

1. **Flat config via `eslint/config` helpers, not a raw array.** Use `defineConfig` and `globalIgnores` from `eslint/config` (present in eslint 9.39). Rationale: this is the official Next 16 migration path and keeps the config readable; alternatives (hand-built arrays with `files`/`ignores` blocks) add ceremony without benefit.
2. **`core-web-vitals` flat entry only.** The `eslint.config.mjs` spreads `...nextVitals` from `eslint-config-next/core-web-vitals`. Rationale: it is the exact rule surface the legacy `.eslintrc.json` used (`@next/eslint-plugin-next` core-web-vitals config plus the standard React/Hooks/a11y sets), so parity is maintained and first-run noise is minimized. Alternative considered — also spreading `eslint-config-next/typescript` (`tseslint` `recommended`) — rejected: new rule surface, extra triage burden, no prior behavior to preserve.
3. **`typescript-eslint@^8.49.0` as a declared devDependency.** The flat config cannot even import without it resolving. It is already in the pnpm store, so `pnpm add -D` re-links without network downloads. Alternative considered — bundling a hand-rolled config that avoids the import — rejected: that would fork us off the maintained path.
4. **Ignores scoped to artifacts only.** `globalIgnores([".next/**", "out/**", "build/**", "test-results/**"])`. Rationale: `.next`/`out`/`build` are build output; `test-results/` is Playwright output now gitignored (pending commit adds it to `.gitignore`). `node_modules` is ignored by ESLint by default. E2E specs remain linted, matching legacy behavior.
5. **Script shape: `"lint": "eslint ."`.** No flags. Rationale: keep the gate honest — a bare run must pass; `--max-warnings` tuning is a policy decision deferred until first-run triage reveals volume.

## Risks / Trade-offs

- [First-run violation noise] → Run `eslint .` once during implementation; fix trivial violations (auto-fixable), record the rest in the task description as a known baseline; do NOT disable rules wholesale.
- [Wrong-rule-diff between eslintrc and flat config] → Mitigated by using the official flat entry, which maps 1:1 to the legacy `next/core-web-vitals` surface; verify by diffing rule counts if suspicious.
- [package.json collision with staged CRM commit] → Land this change as a separate commit after the pending one; if staging mixes, unstage `package.json`/`pnpm-lock.yaml` selectively.
- [ESLint config churn in next minor releases] → Config is two lines of imports + ignores; cost of revisiting is trivial.

## Migration Plan

1. Delete `next_b2b_starter/.eslintrc.json`.
2. Create `next_b2b_starter/eslint.config.mjs` with the flat config above.
3. Run `pnpm add -D typescript-eslint@^8.49.0` (links from store, no downloads).
4. Update `package.json` `lint` script to `"eslint ."`.
5. Verify: `pnpm lint` (exit 0 after triage), `pnpm build`, `npx tsc --noEmit`.
6. Rollback: restore `.eslintrc.json` from git and revert the script change; delete `eslint.config.mjs`. No runtime or Stytch state exists to roll back.

## Open Questions

- None blocking. First-run lint violation volume is unknown but scoped by the triage task.
