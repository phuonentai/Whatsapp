import { test, expect } from "@playwright/test";
import { ContactsPage } from "../page-objects/contacts.page";

test.describe("Contactos", () => {
  let contactsPage: ContactsPage;

  test.beforeEach(async ({ page }) => {
    contactsPage = new ContactsPage(page);
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });
  });

  test("CRUD: create, view, update, delete contact", async ({ page }) => {
    await contactsPage.goto();

    const phone = `+57300${Date.now()}`;
    const name = `Test Contact ${Date.now()}`;

    await contactsPage.create({ phone, name, leadStatus: "nuevo" });
    const row = await contactsPage.getRow(phone);
    expect(row).not.toBeNull();

    await contactsPage.page.fill('input[placeholder*="buscar"]', name);
    await contactsPage.page.waitForTimeout(300);

    const updatedName = `${name} Updated`;
    await contactsPage.page.fill('input[name="display_name"]', updatedName);
    await contactsPage.page.getByRole("button", { name: /guardar/i }).click();
    await contactsPage.page.waitForTimeout(300);

    await contactsPage.delete(phone);
    const deleted = await contactsPage.getRow(phone);
    expect(deleted).toBeNull();
  });

  test("search filters results", async ({ page }) => {
    await contactsPage.goto();
    await contactsPage.search("nonexistent-contact-xyz");
    await page.waitForTimeout(300);
    const rows = await contactsPage.table.locator("tbody tr").count();
    expect(rows).toBe(0);
  });

  test("empty phone shows validation error", async ({ page }) => {
    await contactsPage.goto();
    await contactsPage.newContactButton.click();
    await page.getByRole("button", { name: /guardar|crear/i }).click();
    const error = page.locator("text=requerido,text=obligatorio,text=inválido");
    await expect(error).toBeVisible();
  });
});
