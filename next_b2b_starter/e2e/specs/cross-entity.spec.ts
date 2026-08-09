import { test, expect } from "@playwright/test";

test.describe("Cross-Entity Workflow", () => {
  test("full workflow: company → contact → deal → tag → activity", async ({ page }) => {
    await page.setExtraHTTPHeaders({
      "X-Test-Org-ID": "test-org-enterprise:admin-enterprise@test.com",
    });

    const ts = Date.now();
    const companyName = `WF Company ${ts}`;
    const contactPhone = `+57300${ts}`;
    const contactName = `WF Contact ${ts}`;
    const dealName = `WF Deal ${ts}`;
    const tagName = `WF Tag ${ts}`;
    const activitySubject = `WF Activity ${ts}`;

    // 1. Create company
    await page.goto("/dashboard/crm?view=empresas");
    await page.waitForSelector("table");
    await page.getByRole("button", { name: /nueva empresa/i }).click();
    await page.fill('input[name="name"]', companyName);
    await page.fill('input[name="nit"]', `${ts}`);
    await page.getByRole("button", { name: /guardar|crear/i }).click();
    await page.waitForResponse((res) => res.url().includes("/api/crm/empresas") && res.ok());
    await expect(page.locator(`text=${companyName}`)).toBeVisible();

    // 2. Create contact
    await page.goto("/dashboard/crm?view=contactos");
    await page.waitForSelector("table");
    await page.getByRole("button", { name: /nuevo contacto/i }).click();
    await page.fill('input[name="phone"]', contactPhone);
    await page.fill('input[name="display_name"]', contactName);
    await page.getByRole("button", { name: /guardar|crear/i }).click();
    await page.waitForResponse((res) => res.url().includes("/api/crm/contactos") && res.ok());
    await expect(page.locator(`text=${contactPhone}`)).toBeVisible();

    // 3. Create deal
    await page.goto("/dashboard/crm?view=negocios");
    await page.waitForSelector('[data-testid="kanban-board"]');
    await page.getByRole("button", { name: /nuevo negocio/i }).click();
    await page.fill('input[name="nombre"]', dealName);
    await page.getByRole("button", { name: /guardar|crear/i }).click();
    await page.waitForResponse((res) => res.url().includes("/api/crm/negocios") && res.ok());
    await expect(
      page.locator(`[data-testid="deal-card"]:has-text("${dealName}")`)
    ).toBeVisible();

    // 4. Create tag
    await page.goto("/dashboard/crm?view=etiquetas");
    await page.waitForSelector('[data-testid="tag-list"]');
    await page.getByRole("button", { name: /nueva etiqueta/i }).click();
    await page.fill('input[name="nombre"]', tagName);
    await page.getByRole("button", { name: /guardar|crear/i }).click();
    await page.waitForResponse((res) => res.url().includes("/api/crm/etiquetas") && res.ok());

    // 5. Create activity
    await page.goto("/dashboard/crm?view=actividad");
    await page.waitForSelector('[data-testid="activity-timeline"]');
    await page.getByRole("button", { name: /nueva actividad/i }).click();
    await page.selectOption('select[name="tipo"]', "nota");
    await page.fill('input[name="asunto"]', activitySubject);
    await page.getByRole("button", { name: /guardar|crear/i }).click();
    await page.waitForResponse((res) => res.url().includes("/api/crm/actividades") && res.ok());
    await expect(page.locator(`text=${activitySubject}`)).toBeVisible();
  });

  test("contact detail flow: tag via picker + note activity in timeline", async ({ page }) => {
    await page.setExtraHTTPHeaders({
      "X-Test-Org-ID": "test-org-enterprise:admin-enterprise@test.com",
    });

    const ts = Date.now();
    const contactPhone = `+57302${ts}`;
    const contactName = `Detail Contact ${ts}`;
    const tagName = `Detail Tag ${ts}`;
    const noteSubject = `Detail Note ${ts}`;

    // 1. Create contact
    await page.goto("/dashboard/crm?view=contactos");
    await page.waitForSelector("table");
    await page.getByRole("button", { name: /nuevo contacto/i }).click();
    await page.fill('input[name="phone"]', contactPhone);
    await page.fill('input[name="display_name"]', contactName);
    await page.getByRole("button", { name: /guardar|crear/i }).click();
    await page.waitForResponse((res) => res.url().includes("/api/crm/contactos") && res.ok());

    // 2. Create a tag
    await page.goto("/dashboard/crm?view=etiquetas");
    await page.waitForSelector('[data-testid="tag-list"]');
    await page.getByRole("button", { name: /nueva etiqueta/i }).click();
    await page.fill('input[name="nombre"]', tagName);
    await page.getByRole("button", { name: /guardar|crear/i }).click();
    await page.waitForResponse((res) => res.url().includes("/api/crm/etiquetas") && res.ok());

    // 3. Open contact detail
    await page.goto("/dashboard/crm?view=contactos");
    await page.waitForSelector("table");
    await page.locator(`tr:has-text("${contactPhone}")`).click();
    await page.waitForURL(/view=contactos&id=\d+/);
    await expect(page.locator(`text=${contactName}`)).toBeVisible();

    // 4. Attach tag via the picker
    await page.getByRole("button", { name: /asignar etiqueta/i }).click();
    await page.selectOption('select[aria-label="Seleccionar etiqueta"]', { label: tagName });
    await page.waitForResponse((res) =>
      res.url().includes("/api/crm/etiquetas/entity/contact") && res.ok()
    );
    await expect(page.locator(`text=${tagName}`)).toBeVisible();

    // 5. Add a note from the detail view
    await page.getByRole("button", { name: /agregar nota/i }).click();
    await page.fill('input[placeholder="Asunto"]', noteSubject);
    await page.getByRole("button", { name: /guardar nota/i }).click();
    await page.waitForResponse((res) => res.url().includes("/api/crm/actividades") && res.ok());
    await expect(page.locator(`text=${noteSubject}`)).toBeVisible();
  });
});
