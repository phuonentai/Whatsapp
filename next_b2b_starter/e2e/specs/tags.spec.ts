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

  test("tag a contact and a deal", async ({ page }) => {
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-enterprise:admin-enterprise@test.com" });

    const ts = Date.now();
    const contactPhone = `+57312${ts}`;
    const contactName = `Tagged Contact ${ts}`;
    const dealName = `Tagged Deal ${ts}`;
    const tagName = `Entity Tag ${ts}`;

    // Create tag
    await tagsPage.goto();
    await tagsPage.create({ name: tagName });

    // Create contact
    await page.goto("/dashboard/crm?view=contactos");
    await page.waitForSelector("table");
    await page.getByRole("button", { name: /nuevo contacto/i }).click();
    await page.fill('input[name="phone"]', contactPhone);
    await page.fill('input[name="display_name"]', contactName);
    await page.getByRole("button", { name: /guardar|crear/i }).click();
    await page.waitForResponse((res) => res.url().includes("/api/crm/contactos") && res.ok());

    // Tag contact via detail picker
    await page.locator(`tr:has-text("${contactPhone}")`).click();
    await page.waitForURL(/view=contactos&id=\d+/);
    await page.getByRole("button", { name: /asignar etiqueta/i }).click();
    await page.selectOption('select[aria-label="Seleccionar etiqueta"]', { label: tagName });
    await page.waitForResponse((res) =>
      res.url().includes("/api/crm/etiquetas/entity/contact") && res.ok()
    );
    await expect(page.locator(`text=${tagName}`)).toBeVisible();

    // Create deal and tag it
    await page.goto("/dashboard/crm?view=negocios");
    await page.waitForSelector('[data-testid="kanban-board"]');
    await page.getByRole("button", { name: /nuevo negocio/i }).click();
    await page.fill('input[name="nombre"]', dealName);
    await page.getByRole("button", { name: /guardar|crear/i }).click();
    await page.waitForResponse((res) => res.url().includes("/api/crm/negocios") && res.ok());

    const dealCard = page.locator(`[data-testid="deal-card"]:has-text("${dealName}")`);
    await dealCard.click();
    await page.waitForURL(/view=negocios&id=\d+/);
    await page.getByRole("button", { name: /asignar etiqueta/i }).click();
    await page.selectOption('select[aria-label="Seleccionar etiqueta"]', { label: tagName });
    await page.waitForResponse((res) =>
      res.url().includes("/api/crm/etiquetas/entity/deal") && res.ok()
    );
    await expect(page.locator(`text=${tagName}`)).toBeVisible();
  });

  test("untag an entity removes the tag", async ({ page }) => {
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-enterprise:admin-enterprise@test.com" });

    const ts = Date.now();
    const contactPhone = `+57313${ts}`;
    const contactName = `Untag Contact ${ts}`;
    const tagName = `Untag Tag ${ts}`;

    await tagsPage.goto();
    await tagsPage.create({ name: tagName });

    await page.goto("/dashboard/crm?view=contactos");
    await page.waitForSelector("table");
    await page.getByRole("button", { name: /nuevo contacto/i }).click();
    await page.fill('input[name="phone"]', contactPhone);
    await page.fill('input[name="display_name"]', contactName);
    await page.getByRole("button", { name: /guardar|crear/i }).click();
    await page.waitForResponse((res) => res.url().includes("/api/crm/contactos") && res.ok());

    await page.locator(`tr:has-text("${contactPhone}")`).click();
    await page.waitForURL(/view=contactos&id=\d+/);
    await page.getByRole("button", { name: /asignar etiqueta/i }).click();
    await page.selectOption('select[aria-label="Seleccionar etiqueta"]', { label: tagName });
    await page.waitForResponse((res) =>
      res.url().includes("/api/crm/etiquetas/entity/contact") && res.ok()
    );
    await expect(page.locator(`text=${tagName}`)).toBeVisible();

    // Remove the tag
    await page.locator(`[data-testid="entity-tag"]:has-text("${tagName}")`)
      .locator('button[aria-label="Quitar"]').click();
    await page.waitForResponse((res) => res.url().includes("/api/crm/etiquetas/entity") && res.ok());
    await expect(page.locator(`[data-testid="entity-tag"]:has-text("${tagName}")`)).toHaveCount(0);
  });
});
