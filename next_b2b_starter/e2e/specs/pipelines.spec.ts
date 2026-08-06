import { test, expect } from "@playwright/test";
import { PipelinesPage } from "../page-objects/pipelines.page";

test.describe("Pipelines", () => {
  let pipelinesPage: PipelinesPage;

  test.beforeEach(async ({ page }) => {
    pipelinesPage = new PipelinesPage(page);
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });
  });

  test("default pipeline shows with stages", async ({ page }) => {
    await pipelinesPage.goto();
    const defaultPipeline = await pipelinesPage.getPipeline("Pipeline de Ventas");
    expect(defaultPipeline).not.toBeNull();

    const stages = page.locator('[data-testid="stage-item"]');
    const count = await stages.count();
    expect(count).toBeGreaterThanOrEqual(4);
  });

  test("create pipeline with stages", async ({ page }) => {
    await pipelinesPage.goto();

    const name = `Test Pipeline ${Date.now()}`;
    await pipelinesPage.create({
      name,
      stages: [
        { name: "Contactado", color: "#3B82F6" },
        { name: "Cerrado", color: "#10B981" },
      ],
    });

    const pipeline = await pipelinesPage.getPipeline(name);
    expect(pipeline).not.toBeNull();
  });
});
