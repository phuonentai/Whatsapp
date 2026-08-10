import { Page, Locator, expect } from "@playwright/test";

export class DealsKanbanPage {
  readonly page: Page;
  readonly newDealButton: Locator;
  readonly board: Locator;

  constructor(page: Page) {
    this.page = page;
    this.newDealButton = page.getByRole("button", { name: /nuevo negocio/i });
    this.board = page.locator('[data-testid="kanban-board"]');
  }

  async goto() {
    await this.page.goto("/dashboard/crm?view=negocios");
    await this.page.waitForSelector('[data-testid="kanban-board"]');
  }

  async create(data: { name: string; amount?: string; contactId?: number; companyId?: number }) {
    await this.newDealButton.click();
    await this.page.fill('input[name="nombre"]', data.name);
    if (data.amount) await this.page.fill('input[name="monto"]', data.amount);
    await this.page.getByRole("button", { name: /guardar|crear/i }).click();
    await this.page.waitForResponse((res) => res.url().includes("/api/crm/negocios") && res.request().method() === "POST" && res.ok());
  }

  async getCard(name: string): Promise<Locator | null> {
    const card = this.board.locator(`[data-testid="deal-card"]:has-text("${name}")`);
    try {
      await card.first().waitFor({ state: "attached", timeout: 3000 });
      return card;
    } catch {
      return null;
    }
  }

  async moveToStage(dealName: string, targetStage: string) {
    const card = await this.getCard(dealName);
    if (!card) throw new Error(`Deal ${dealName} not found`);
    // Use the card's "Mover a..." <select>, which triggers the same onMove →
    // PUT /negocios/:id/etapa handler as a dnd-kit drag. Headless pointer drag
    // against dnd-kit's PointerSensor is unreliable in Playwright.
    const moveDone = this.page.waitForResponse(
      (res) =>
        res.url().includes("/api/crm/negocios/") &&
        res.url().includes("/etapa") &&
        res.request().method() === "PUT" &&
        res.ok()
    );
    await card.locator("select").selectOption({ label: targetStage });
    await moveDone;
    await expect(
      this.board
        .locator(`[data-testid="stage-column"]:has(h3:text-is("${targetStage}"))`)
        .locator(`[data-testid="deal-card"]:has-text("${dealName}")`)
    ).toBeVisible();
  }

  async delete(name: string) {
    const card = await this.getCard(name);
    if (!card) throw new Error(`Deal ${name} not found`);
    await card.locator('button[aria-label="Eliminar"]').click();
    await this.page.getByRole("button", { name: /confirmar|sí|eliminar/i }).click();
  }
}
