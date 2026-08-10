import { test, expect } from "@playwright/test";
import { SettingsPage } from "../page-objects/settings.page";
import { apiRequest } from "../helpers/api";

test.describe("Settings UI", () => {
  test.beforeEach(async ({ page }) => {
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });
  });

  test("invite member form renders with role selector", async ({ page }) => {
    const settings = new SettingsPage(page);
    await settings.goto();
    const dialog = await settings.openInviteDialog();
    await expect(dialog.getByLabel(/email/i)).toBeVisible();
    await expect(dialog.getByRole("combobox", { name: "Role" })).toBeVisible();
  });

  test("modules section lists modules", async ({ page }) => {
    const settings = new SettingsPage(page);
    await settings.goto();
    await settings.openSection("Modules");
    await settings.assertModuleNameVisible("Tickets (Helpdesk)");
  });

  test("playbook setup card renders on overview", async ({ page }) => {
    const settings = new SettingsPage(page);
    await settings.goto();
    await settings.assertPlaybookVisible("Comercio");
  });

  test("profile section allows editing workspace name", async ({ page }) => {
    const settings = new SettingsPage(page);
    await settings.goto();
    await settings.editWorkspaceName(`E2E Workspace ${Date.now()}`);
  });

  test("subscription tab shows the plan", async ({ page }) => {
    const settings = new SettingsPage(page);
    await settings.goto();
    await settings.openSection("Subscription & billing");
    await settings.assertPlanVisible("Pro");
  });

  test("whatsapp config section renders", async ({ page }) => {
    const settings = new SettingsPage(page);
    await settings.goto();
    await settings.openSection("Messaging");
    await settings.assertWhatsappConfigVisible();
  });

  test("member list shows roles with role controls", async ({ page }) => {
    const settings = new SettingsPage(page);
    await settings.goto();
    await settings.openSection("Team access");
    await settings.assertMemberRole("admin-pro@test.com");
  });

  test("audit log section renders", async ({ page }) => {
    const settings = new SettingsPage(page);
    await settings.goto();
    await settings.openSection("Audit log");
    await expect(page.getByTestId("audit-log-list")).toBeVisible();
  });

  test("member sees no invite form in team access", async ({ page }) => {
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:member-pro@test.com" });
    const settings = new SettingsPage(page);
    await settings.goto();
    // Admin-only sections are hidden entirely from non-admin members.
    await expect(page.getByRole("button", { name: /team access/i })).not.toBeVisible();
    await expect(page.getByRole("button", { name: /add member/i })).not.toBeVisible();
  });
});
