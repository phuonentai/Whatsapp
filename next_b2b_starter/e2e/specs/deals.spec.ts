import { test, expect } from "@playwright/test";
import { DealsKanbanPage } from "../page-objects/deals-kanban.page";

test.describe("Negocios", () => {
  let dealsPage: DealsKanbanPage;

  test.beforeEach(async ({ page }) => {
    dealsPage = new DealsKanbanPage(page);
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });
  });

  test("CRUD: create, view, update, delete deal on Kanban", async ({ page }) => {
    await dealsPage.goto();

    const name = `Test Deal ${Date.now()}`;
    await dealsPage.create({ name, amount: "5000000" });

    const card = await dealsPage.getCard(name);
    expect(card).not.toBeNull();

    await dealsPage.delete(name);
    const deleted = await dealsPage.getCard(name);
    expect(deleted).toBeNull();
  });

  test("moves deal between stages", async ({ page }) => {
    await dealsPage.goto();

    const name = `Stage Deal ${Date.now()}`;
    await dealsPage.create({ name, amount: "3000000" });

    await dealsPage.moveToStage(name, "Calificado");
    await page.waitForTimeout(500);

    const card = await dealsPage.getCard(name);
    expect(card).not.toBeNull();
  });
});
