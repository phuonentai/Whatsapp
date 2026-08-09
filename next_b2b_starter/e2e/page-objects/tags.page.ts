import { Page, Locator } from "@playwright/test";

export class TagsPage {
  readonly page: Page;
  readonly newTagButton: Locator;
  readonly tagList: Locator;

  constructor(page: Page) {
    this.page = page;
    this.newTagButton = page.getByRole("button", { name: /nueva etiqueta/i });
    this.tagList = page.locator('[data-testid="tag-list"]');
  }

  async goto() {
    await this.page.goto("/dashboard/crm?view=etiquetas");
    await this.page.waitForSelector('[data-testid="tag-list"]');
  }

  async create(data: { name: string; color?: string }) {
    await this.newTagButton.click();
    await this.page.fill('input[name="nombre"]', data.name);
    if (data.color) await this.page.fill('input[name="color"]', data.color);
    await this.page.getByRole("button", { name: /guardar|crear/i }).click();
    await this.page.waitForResponse((res) => res.url().includes("/api/crm/etiquetas") && res.ok());
  }

  async getTag(name: string): Promise<Locator | null> {
    const tag = this.tagList.locator(`text="${name}"`);
    try {
      await tag.first().waitFor({ state: "attached", timeout: 3000 });
      return tag;
    } catch {
      return null;
    }
  }

  async delete(name: string) {
    const tag = await this.getTag(name);
    if (!tag) throw new Error(`Tag ${name} not found`);
    await tag.locator('button[aria-label="Eliminar"]').click();
    await this.page.getByRole("button", { name: /confirmar|sí|eliminar/i }).click();
  }

  async tagEntity(entityType: string, entityId: number, tagName: string) {
    await this.page.goto(`/dashboard/crm?view=${entityType}&id=${entityId}`);
    await this.page.getByRole("button", { name: /etiquetar/i }).click();
    await this.page.fill('input[data-testid="tag-picker"]', tagName);
    await this.page.getByRole("option", { name: tagName }).click();
  }
}
