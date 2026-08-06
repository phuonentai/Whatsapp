import { test, expect } from "@playwright/test";
import { TagsPage } from "../page-objects/tags.page";

test.describe("Etiquetas", () => {
  let tagsPage: TagsPage;

  test.beforeEach(async ({ page }) => {
    tagsPage = new TagsPage(page);
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-enterprise:admin-enterprise@test.com" });
  });

  test("create and delete tag", async ({ page }) => {
    await tagsPage.goto();

    const name = `Test Tag ${Date.now()}`;
    await tagsPage.create({ name, color: "#FF0000" });

    const tag = await tagsPage.getTag(name);
    expect(tag).not.toBeNull();

    await tagsPage.delete(name);
    const deleted = await tagsPage.getTag(name);
    expect(deleted).toBeNull();
  });

  test("duplicate tag name shows error", async ({ page }) => {
    await tagsPage.goto();
    const name = `Dup Tag ${Date.now()}`;
    await tagsPage.create({ name });
    await tagsPage.newTagButton.click();
    await page.fill('input[name="nombre"]', name);
    await page.getByRole("button", { name: /guardar|crear/i }).click();
    const error = page.locator("text=ya existe,text=duplicado");
    await expect(error).toBeVisible();
  });
});
