import { Page, Locator } from "@playwright/test";

export class ActivitiesPage {
  readonly page: Page;
  readonly newActivityButton: Locator;
  readonly timeline: Locator;

  constructor(page: Page) {
    this.page = page;
    this.newActivityButton = page.getByRole("button", { name: /nueva actividad/i });
    this.timeline = page.locator('[data-testid="activity-timeline"]');
  }

  async goto() {
    await this.page.goto("/dashboard/crm?view=actividad");
    await this.page.waitForSelector('[data-testid="activity-timeline"]');
  }

  async create(data: { type: string; subject: string; content?: string }) {
    await this.newActivityButton.click();
    await this.page.selectOption('select[name="tipo"]', data.type);
    await this.page.fill('input[name="asunto"]', data.subject);
    if (data.content) await this.page.fill('textarea[name="contenido"]', data.content);
    await this.page.getByRole("button", { name: /guardar|crear/i }).click();
    await this.page.waitForResponse((res) => res.url().includes("/api/crm/actividades") && res.ok());
  }

  async getActivity(subject: string): Promise<Locator | null> {
    const item = this.timeline.locator(`[data-testid="activity-item"]:has-text("${subject}")`);
    return (await item.count()) > 0 ? item : null;
  }

  async filterByType(type: string) {
    await this.page.selectOption('select[data-testid="activity-type-filter"]', type);
    await this.page.waitForTimeout(300);
  }
}
