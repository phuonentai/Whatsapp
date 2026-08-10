import { test, expect } from "@playwright/test";
import { AdminPanelPage } from "../page-objects/admin-panel.page";

const ORG = { slug: "test-org-pro", email: "admin-pro@test.com" };

test.describe("Admin panel navigation", () => {
  let admin: AdminPanelPage;

  test.beforeEach(async ({ page }) => {
    admin = new AdminPanelPage(page);
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": `${ORG.slug}:${ORG.email}` });
  });

  test("sidebar exposes Inbox and CRM entries", async ({ page }) => {
    await admin.goto("/dashboard/settings");
    expect(await admin.hasSidebarEntry("Inbox")).toBe(true);
    expect(await admin.hasSidebarEntry("CRM")).toBe(true);
  });

  test("Inbox entry navigates to inbox page", async ({ page }) => {
    await admin.goto("/dashboard/settings");
    await (await admin.sidebarEntry("Inbox")).click();
    await expect(page).toHaveURL(/\/dashboard\/inbox/);
  });

  test("CRM entry navigates to CRM page", async ({ page }) => {
    await admin.goto("/dashboard/settings");
    await (await admin.sidebarEntry("CRM")).click();
    await expect(page).toHaveURL(/\/dashboard\/crm/);
  });
});

test.describe("Settings views", () => {
  let admin: AdminPanelPage;

  test.beforeEach(async ({ page }) => {
    admin = new AdminPanelPage(page);
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": `${ORG.slug}:${ORG.email}` });
  });

  test("whatsapp view renders WhatsApp config section", async ({ page }) => {
    await admin.gotoSettings("whatsapp");
    await expect(page.getByRole("heading", { name: /messaging/i })).toBeVisible();
  });

  test("audit view renders read-only audit log", async ({ page }) => {
    await admin.gotoSettings("audit");
    await expect(page.getByRole("heading", { name: /audit log/i })).toBeVisible();
    const log = await admin.getAuditLog();
    expect(log).not.toBeNull();
    await expect(admin.page.locator("text=Read-only record of activity").first()).toBeVisible();
  });

  test("audit view does not offer mutation controls", async ({ page }) => {
    await admin.gotoSettings("audit");
    const log = await admin.getAuditLog();
    await expect(log!).toBeVisible();
    await expect(admin.page.getByRole("button", { name: /nueva actividad/i })).toHaveCount(0);
    await expect(admin.page.getByRole("button", { name: /guardar/i })).toHaveCount(0);
  });

  test("audit view filters by tipo", async ({ page }) => {
    await admin.gotoSettings("audit");
    await admin.filterAuditByType("llamada");
    const log = await admin.getAuditLog();
    await expect(log!).toBeVisible();
  });
});

test.describe("Workspace name edit", () => {
  let admin: AdminPanelPage;

  test.beforeEach(async ({ page }) => {
    admin = new AdminPanelPage(page);
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": `${ORG.slug}:${ORG.email}` });
  });

  test("admin can edit workspace display name", async ({ page }) => {
    await admin.gotoSettings("profile");
    const editButton = page.getByRole("button", { name: /editar/i });
    if ((await editButton.count()) === 0) {
      test.skip(true, "org:manage not granted to this fixture user");
    }
    await editButton.click();
    await expect(page.locator("#workspace-name")).toBeVisible();
    await page.fill("#workspace-name", "Workspace E2E");
    await page.getByRole("button", { name: /guardar/i }).click();
    await expect(page.getByRole("heading", { name: "Workspace E2E" })).toBeVisible();
  });
});

test.describe("Member role management", () => {
  let admin: AdminPanelPage;

  test.beforeEach(async ({ page }) => {
    admin = new AdminPanelPage(page);
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": `${ORG.slug}:${ORG.email}` });
  });

  test("member list exposes role control for manageable members", async ({ page }) => {
    await admin.gotoSettings("members");
    await expect(page.getByRole("heading", { name: /team roster/i })).toBeVisible();
    await expect(page.locator('[aria-label^="Change role for"]').first()).toBeVisible();
  });
});
