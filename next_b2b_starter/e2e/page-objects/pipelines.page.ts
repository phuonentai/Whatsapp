import { Page, Locator } from "@playwright/test";

export class PipelinesPage {
  readonly page: Page;
  readonly newPipelineButton: Locator;
  readonly pipelineList: Locator;

  constructor(page: Page) {
    this.page = page;
    this.newPipelineButton = page.getByRole("button", { name: /nuevo pipeline/i });
    this.pipelineList = page.locator('[data-testid="pipeline-list"]');
  }

  async goto() {
    await this.page.goto("/dashboard/crm?view=pipelines");
    await this.page.waitForSelector('[data-testid="pipeline-list"]');
  }

  async create(data: { name: string; stages: { name: string; color: string }[] }) {
    await this.newPipelineButton.click();
    await this.page.fill('input[name="nombre"]', data.name);
    for (const stage of data.stages) {
      await this.page.getByRole("button", { name: /agregar etapa/i }).click();
      await this.page.fill('input[name="stage_name"]:last', stage.name);
      await this.page.fill('input[name="stage_color"]:last', stage.color);
    }
    await this.page.getByRole("button", { name: /guardar|crear/i }).click();
  }

  async getPipeline(name: string): Promise<Locator | null> {
    const item = this.pipelineList.locator(`text="${name}"`);
    return (await item.count()) > 0 ? item : null;
  }

  async editStage(stageName: string, newName: string) {
    const stage = this.page.locator(`[data-testid="stage-item"]:has-text("${stageName}")`);
    await stage.locator('button[aria-label="Editar"]').click();
    await this.page.fill('input[name="stage_name"]', newName);
    await this.page.getByRole("button", { name: /guardar/i }).click();
  }
}
