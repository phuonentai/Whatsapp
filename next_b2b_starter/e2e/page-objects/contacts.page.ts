import { Page, Locator } from "@playwright/test";

export class ContactsPage {
  readonly page: Page;
  readonly newContactButton: Locator;
  readonly searchInput: Locator;
  readonly table: Locator;

  constructor(page: Page) {
    this.page = page;
    this.newContactButton = page.getByRole("button", { name: /nuevo contacto/i });
    this.searchInput = page.getByPlaceholder(/buscar/i);
    this.table = page.locator("table");
  }

  async goto() {
    await this.page.goto("/dashboard/crm?view=contactos");
    await this.page.waitForSelector("table");
  }

  async create(data: {
    phone: string;
    name?: string;
    email?: string;
    leadStatus?: string;
  }) {
    await this.newContactButton.click();
    await this.page.fill('input[name="phone"]', data.phone);
    if (data.name) await this.page.fill('input[name="display_name"]', data.name);
    if (data.email) await this.page.fill('input[name="email"]', data.email);
    if (data.leadStatus) {
      await this.page.selectOption('select[name="lead_status"]', data.leadStatus);
    }
    await this.page.getByRole("button", { name: /guardar|crear/i }).click();
    await this.page.waitForResponse((res) => res.url().includes("/api/crm/contactos") && res.ok());
  }

  async search(query: string) {
    await this.searchInput.fill(query);
    await this.page.waitForTimeout(500);
  }

  async getRow(phone: string): Promise<Locator | null> {
    const row = this.table.locator(`tr:has-text("${phone}")`);
    try {
      await row.first().waitFor({ state: "attached", timeout: 3000 });
      return row;
    } catch {
      return null;
    }
  }

  async delete(phone: string) {
    const row = await this.getRow(phone);
    if (!row) throw new Error(`Contact ${phone} not found`);
    await row.locator('button:has-text("eliminar"), button[aria-label="Eliminar"]').click();
    await this.page.getByRole("button", { name: /confirmar|sí|eliminar/i }).click();
  }
}
