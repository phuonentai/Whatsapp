import { test, expect } from "@playwright/test";
import { CompaniesPage } from "../page-objects/companies.page";

test.describe("Empresas", () => {
  let companiesPage: CompaniesPage;

  test.beforeEach(async ({ page }) => {
    companiesPage = new CompaniesPage(page);
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });
  });

  test("CRUD: create, view, update, delete company", async ({ page }) => {
    await companiesPage.goto();

    const name = `Test Company ${Date.now()}`;
    const nit = `${Date.now()}`;

    await companiesPage.create({ name, nit, sector: "Tecnología", ciudad: "Bogotá" });
    const row = await companiesPage.getRow(name);
    expect(row).not.toBeNull();

    await page.fill('input[name="name"]', `${name} Updated`);
    await page.getByRole("button", { name: /guardar/i }).click();
    await page.waitForTimeout(300);

    await companiesPage.delete(name);
    const deleted = await companiesPage.getRow(name);
    expect(deleted).toBeNull();
  });

  test("search filters results", async ({ page }) => {
    await companiesPage.goto();
    await companiesPage.search("nonexistent-company-xyz");
    await page.waitForTimeout(300);
    const rows = await companiesPage.table.locator("tbody tr").count();
    expect(rows).toBe(0);
  });

  test("duplicate name shows error", async ({ page }) => {
    await companiesPage.goto();
    const name = `Dup Company ${Date.now()}`;
    await companiesPage.create({ name, nit: `${Date.now()}` });
    await companiesPage.newCompanyButton.click();
    await page.fill('input[name="name"]', name);
    await page.getByRole("button", { name: /guardar|crear/i }).click();
    const error = page.locator("text=ya existe,text=duplicado");
    await expect(error).toBeVisible();
  });
});
