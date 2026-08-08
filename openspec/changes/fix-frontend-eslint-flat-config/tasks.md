## 1. Config Migration

- [x] 1.1 [FE-NEXT] Delete `next_b2b_starter/.eslintrc.json`
- [x] 1.2 [FE-NEXT] Create `next_b2b_starter/eslint.config.mjs`: `import { defineConfig, globalIgnores } from "eslint/config"`, spread `eslint-config-next/core-web-vitals` flat config, `globalIgnores([".next/**", "out/**", "build/**", "test-results/**"])`

## 2. Dependency & Script

- [x] 2.1 [FE-NEXT] Run `pnpm add -D typescript-eslint@^8.49.0` (re-links from pnpm store 8.49.0, no downloads) and verify it resolves at top level (`node -e "require.resolve('typescript-eslint')"`)
- [x] 2.2 [FE-NEXT] Change `package.json` `lint` script from `PORT=0 next lint` to `eslint .`

## 3. Verification

- [x] 3.1 [FE-NEXT] `pnpm lint` runs with the flat config (exits 0 after triage) — triage outcome: 2 trivial `react/no-unescaped-entities` fixed in `app/auth/page.tsx:381` and `components/auth/permission-gate.tsx:74`. **Known baseline (13 errors, 1 warning) recorded, out of scope** per design Non-Goals (behavior-affecting refactors): `rules-of-hooks` ×4 in `app/dashboard/inbox/page.tsx:38-43`; `set-state-in-effect` ×9 in `knowledge-content.tsx:54,60`, `settings-content.tsx:278,360,366`, `modules-section.tsx:49`, `whatsapp-config-section.tsx:48`, `plans-modal.tsx:35`, `auth-context.tsx:194`; `exhaustive-deps` warning in `deal-kanban.tsx:166`. Baseline clearing tracked as a follow-up change (per amended delta spec). Lint now exits 1 with only the documented baseline.
- [x] 3.2 [FE-NEXT] `pnpm build` still green — verified "Compiled successfully in 54s" after config migration
- [x] 3.3 [FE-NEXT] `npx tsc --noEmit -p tsconfig.json` exits with zero errors — verified TSC_EXIT=0
- [ ] 3.4 [FE-NEXT] Commit this change as a separate commit after the pending staged CRM commit lands; do not mix `package.json`/`pnpm-lock.yaml` with the CRM stage
- [ ] 3.5 [FE-NEXT] Record archive decision per governance-workflow: either run the archive workflow or append an explicit "Archive deferred: <reason>" entry
