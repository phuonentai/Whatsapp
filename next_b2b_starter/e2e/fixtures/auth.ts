import { Page } from "@playwright/test";

const TEST_ORGS = {
  free: { slug: "test-org-free", email: "admin-free@test.com" },
  pro: { slug: "test-org-pro", email: "admin-pro@test.com" },
  enterprise: { slug: "test-org-enterprise", email: "admin-enterprise@test.com" },
  rbacAdmin: { slug: "test-org-rbac", email: "admin-rbac@test.com" },
  rbacManager: { slug: "test-org-rbac", email: "manager-rbac@test.com" },
  rbacMember: { slug: "test-org-rbac", email: "member-rbac@test.com" },
};

export type OrgType = keyof typeof TEST_ORGS;

export function getMockAuthHeader(orgType: OrgType): string {
  const org = TEST_ORGS[orgType];
  return `${org.slug}:${org.email}`;
}

export async function loginAs(page: Page, orgType: OrgType): Promise<void> {
  const authHeader = getMockAuthHeader(orgType);
  await page.goto("/dashboard/crm");

  await page.evaluate((header) => {
    document.cookie = `X-Test-Org-ID=${header}; path=/; max-age=3600`;
  }, authHeader);
}

export async function setMockAuthHeader(page: Page, orgType: OrgType): Promise<void> {
  const authHeader = getMockAuthHeader(orgType);
  await page.setExtraHTTPHeaders({ "X-Test-Org-ID": authHeader });
}
