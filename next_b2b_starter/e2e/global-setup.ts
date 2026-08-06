import { FullConfig } from "@playwright/test";

async function globalSetup(config: FullConfig) {
  console.log("Running global setup...");
  console.log("Ensure test database is migrated and seeded before running tests.");
  console.log("Expected test orgs:");
  console.log("  - test-org-free (Free plan)");
  console.log("  - test-org-pro (Pro plan)");
  console.log("  - test-org-enterprise (Enterprise plan)");
  console.log("  - test-org-rbac (Pro plan, admin/manager/member accounts)");
  console.log("Global setup complete.");
}

export default globalSetup;
