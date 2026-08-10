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

    await row!.locator('button:has-text("Editar"), button[aria-label="Editar"]').click();
    await page.fill('input[name="name"]', `${name} Updated`);
    await page.getByRole("button", { name: /guardar/i }).click();
    await expect(page.locator(`tr:has-text("${name} Updated")`)).toBeVisible();

    await companiesPage.delete(name);
    await expect(companiesPage.table.locator(`tr:has-text("${name}")`)).toHaveCount(0);
  });

  test("search filters results", async () => {
    await companiesPage.goto();
    await companiesPage.search("nonexistent-company-xyz");
    await expect(companiesPage.table.locator("tbody tr")).toHaveCount(0);
  });

  test("duplicate name shows error", async ({ page }) => {
    await companiesPage.goto();
    const name = `Dup Company ${Date.now()}`;
    await companiesPage.create({ name, nit: `${Date.now()}` });
    await companiesPage.newCompanyButton.click();
    await page.fill('input[name="name"]', name);
    await page.getByRole("button", { name: /guardar|crear/i }).click();
    const error = page.locator("text=ya existe");
    await expect(error).toBeVisible();
  });

  test("row click opens company detail view", async ({ page }) => {
    await companiesPage.goto();

    const name = `Detail Company ${Date.now()}`;
    await companiesPage.create({ name, nit: `${Date.now()}` });

    const row = await companiesPage.getRow(name);
    await row!.click();
    await page.waitForURL(/view=empresas&id=\d+/);

    await expect(page.locator(`text=${name}`)).toBeVisible();
    await expect(page.getByRole("button", { name: /volver/i })).toBeVisible();

    await page.getByRole("button", { name: /volver/i }).click();
    await page.waitForURL(/view=empresas$/);
  });
});
