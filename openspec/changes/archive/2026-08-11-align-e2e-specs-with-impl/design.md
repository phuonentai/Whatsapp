# Design

## Feature-gating: disabled tab pattern

Verified against `next_b2b_starter/app/dashboard/crm/crm-page.tsx`:

- `TAB_LABELS` maps each tab to a `feature` key and `upgradePlan`; `contactos` is always enabled.
- Tabs are filtered out only when entitlement data explicitly says `false` while loading (absent entitlement keeps them, disabled). Once loaded, each gated tab renders as a `<button disabled={tab.disabled}>` with a plan hint (e.g. `(Pro)` / `Desbloquear con Enterprise`).
- Restricted views render an upgrade banner in the content area.

The e2e test `e2e/specs/feature-gating.spec.ts` asserts `getAttribute("disabled")` on the Empresas/Negocios buttons for `test-org-free`, Pro-tab visibility for `test-org-pro`, all tabs for `test-org-enterprise`, and API 403 via `POST /api/crm/empresas`.

Decision: keep the implemented pattern; amend the spec wording from "SHALL NOT be visible" to "SHALL render disabled (visible but non-interactive, with upgrade hint)". This is a spec correction, not a behavior change.

## Test infrastructure: seeding location

Verified:

- `go-b2b-starter/cmd/seed-e2e/main.go` upserts orgs `test-org-free` (free), `test-org-pro` (pro, admin + member), `test-org-enterprise` (enterprise), `test-org-rbac` (pro, admin/manager/member), plus subscriptions, quotas, and `agent.agent_settings.kill_switch=true`.
- `go-b2b-starter/scripts/run_e2e.sh` resets `saas_db_test`, applies migrations, runs `seed-e2e`, boots backend `:8080` + frontend `:3001`, then runs Playwright.
- GitHub `.github/workflows/ci.yml` and `.gitlab-ci.yml` both run migrate → seed → boot → `pnpm test:e2e`.
- `next_b2b_starter/e2e/global-setup.ts` is a logging stub; `playwright.config.ts` does not reference a `webServer`/`globalSetup` that seeds.

Decision: seeding belongs to the bootstrapping layer (`make test-e2e` / CI), which owns the deterministic DB reset. `global-setup.ts` may validate preconditions but must not create orgs. Amend the spec requirement and scenario accordingly.
