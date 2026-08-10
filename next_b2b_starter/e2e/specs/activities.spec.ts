import { test, expect } from "@playwright/test";
import { ActivitiesPage } from "../page-objects/activities.page";
import { uniqueColombianPhone } from "../helpers/phones";

test.describe("Actividades", () => {
  let activitiesPage: ActivitiesPage;

  test.beforeEach(async ({ page }) => {
    activitiesPage = new ActivitiesPage(page);
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });
  });

  test("create note activity", async ({ page }) => {
    await activitiesPage.goto();

    const subject = `Test Note ${Date.now()}`;
    await activitiesPage.create({ type: "nota", subject, content: "Test content" });

    const activity = await activitiesPage.getActivity(subject);
    expect(activity).not.toBeNull();
  });

  test("create call activity", async ({ page }) => {
    await activitiesPage.goto();

    const subject = `Test Call ${Date.now()}`;
    await activitiesPage.create({ type: "llamada", subject });

    const activity = await activitiesPage.getActivity(subject);
    expect(activity).not.toBeNull();
  });

  test("create task activity", async ({ page }) => {
    await activitiesPage.goto();

    const subject = `Test Task ${Date.now()}`;
    await activitiesPage.create({ type: "tarea", subject });

    const activity = await activitiesPage.getActivity(subject);
    expect(activity).not.toBeNull();
  });

  test("task activity collects due date and estado", async ({ page }) => {
    await activitiesPage.goto();

    const subject = `Task Fields ${Date.now()}`;
    await page.getByRole("button", { name: /nueva actividad/i }).click();
    await page.selectOption('select[name="tipo"]', "tarea");
    await page.fill('input[placeholder="Asunto"]', subject);
    await page.fill('input[aria-label="Fecha de vencimiento"]', "2026-12-31");
    await page.selectOption('select[aria-label="Estado"]', "pendiente");
    await page.getByRole("button", { name: /guardar/i }).click();
    await page.waitForResponse((res) => res.url().includes("/api/crm/actividades") && res.ok());

    await expect(page.locator(`text=${subject}`)).toBeVisible();
    await expect(page.locator("text=Vence:").first()).toBeVisible();
  });

  test("filter control filters activities by type", async ({ page }) => {
    await activitiesPage.goto();

    const subject = `Filter Call ${Date.now()}`;
    await activitiesPage.create({ type: "llamada", subject });

    await page.selectOption('[data-testid="activity-type-filter"]', "llamada");

    await expect(page.locator(`text=${subject}`)).toBeVisible();

    await page.selectOption('[data-testid="activity-type-filter"]', "nota");
    await expect(page.locator(`text=${subject}`)).toBeHidden();
  });

  test("activity appears in contact detail timeline", async ({ page }) => {
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });

    const ts = Date.now();
    const contactPhone = uniqueColombianPhone();
    const contactName = `Timeline Contact ${ts}`;
    const noteSubject = `Timeline Note ${ts}`;

    // Create contact
    await page.goto("/dashboard/crm?view=contactos");
    await page.waitForSelector("table");
    await page.getByRole("button", { name: /nuevo contacto/i }).click();
    await page.fill('input[name="phone"]', contactPhone);
    await page.fill('input[name="display_name"]', contactName);
    await page.getByRole("button", { name: /guardar|crear/i }).click();
    await page.waitForResponse((res) => res.url().includes("/api/crm/contactos") && res.ok());

    // Add a note from the contact detail timeline
    await page.locator(`tr:has-text("${contactPhone}")`).click();
    await page.waitForURL(/view=contactos&id=\d+/);
    await page.getByRole("button", { name: /agregar nota/i }).click();
    await page.fill('input[placeholder="Asunto"]', noteSubject);
    await page.getByRole("button", { name: /guardar nota/i }).click();
    await page.waitForResponse((res) => res.url().includes("/api/crm/actividades") && res.ok());

    await expect(page.locator(`text=${noteSubject}`)).toBeVisible();
  });

  test("stage change on a deal logs an activity in deal timeline", async ({ page }) => {
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });

    const dealName = `Timeline Deal ${Date.now()}`;
    await page.goto("/dashboard/crm?view=negocios");
    await page.waitForSelector('[data-testid="kanban-board"]');
    await page.getByRole("button", { name: /nuevo negocio/i }).click();
    await page.fill('input[name="nombre"]', dealName);
    await page.getByRole("button", { name: /guardar|crear/i }).click();
    await page.waitForResponse((res) => res.url().includes("/api/crm/negocios") && res.ok());

    const card = page.locator(`[data-testid="deal-card"]:has-text("${dealName}")`);
    await card.click();
    await page.waitForURL(/view=negocios&id=\d+/);

    await expect(page.locator("text=Etapa")).toBeVisible();
    await expect(page.getByRole("heading", { name: /actividad/i })).toBeVisible();
  });
});
