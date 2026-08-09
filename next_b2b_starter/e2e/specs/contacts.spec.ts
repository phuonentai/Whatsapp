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
    const editRow = await contactsPage.getRow(phone);
    await editRow!.locator('button:has-text("Editar"), button[aria-label="Editar"]').click();
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

  test("row click opens contact detail view", async ({ page }) => {
    await contactsPage.goto();

    const phone = `+57301${Date.now()}`;
    const name = `Detail Contact ${Date.now()}`;
    await contactsPage.create({ phone, name });

    const row = await contactsPage.getRow(phone);
    await row!.click();
    await page.waitForURL(/view=contactos&id=\d+/);

    await expect(page.locator(`text=${name}`)).toBeVisible();
    await expect(page.getByRole("button", { name: /agregar nota/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /crear negocio/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /editar/i })).toBeVisible();

    await page.getByRole("button", { name: /volver/i }).click();
    await page.waitForURL(/view=contactos$/);
  });
});
