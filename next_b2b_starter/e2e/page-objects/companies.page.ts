import { Page, Locator } from "@playwright/test";

export class CompaniesPage {
  readonly page: Page;
  readonly newCompanyButton: Locator;
  readonly searchInput: Locator;
  readonly table: Locator;

  constructor(page: Page) {
    this.page = page;
    this.newCompanyButton = page.getByRole("button", { name: /nueva empresa/i });
    this.searchInput = page.getByPlaceholder(/buscar/i);
    this.table = page.locator("table");
  }

  async goto() {
    await this.page.goto("/dashboard/crm?view=empresas");
    await this.page.waitForSelector("table");
  }

  async create(data: { name: string; nit?: string; sector?: string; ciudad?: string }) {
    await this.newCompanyButton.click();
    await this.page.fill('input[name="name"]', data.name);
    if (data.nit) await this.page.fill('input[name="nit"]', data.nit);
    if (data.sector) await this.page.fill('input[name="sector"]', data.sector);
    if (data.ciudad) await this.page.fill('input[name="ciudad"]', data.ciudad);
    await this.page.getByRole("button", { name: /guardar|crear/i }).click();
    await this.page.waitForResponse((res) => res.url().includes("/api/crm/empresas") && res.ok());
  }

  async search(query: string) {
    await this.searchInput.fill(query);
    await this.page.waitForTimeout(500);
  }

  async getRow(name: string): Promise<Locator | null> {
    const row = this.table.locator(`tr:has-text("${name}")`);
    try {
      await row.first().waitFor({ state: "attached", timeout: 3000 });
      return row;
    } catch {
      return null;
    }
  }

  async delete(name: string) {
    const row = await this.getRow(name);
    if (!row) throw new Error(`Company ${name} not found`);
    await row.locator('button:has-text("eliminar"), button[aria-label="Eliminar"]').click();
    await this.page.getByRole("button", { name: /confirmar|sí|eliminar/i }).click();
  }
}
