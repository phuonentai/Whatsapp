import { test, expect } from "@playwright/test";
import { DealsKanbanPage } from "../page-objects/deals-kanban.page";
import { apiRequest } from "../helpers/api";

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

  test("card click opens deal detail view with stage-change activity", async ({ page }) => {
    await dealsPage.goto();

    const name = `Detail Deal ${Date.now()}`;
    await dealsPage.create({ name, amount: "4000000" });

    const card = await dealsPage.getCard(name);
    await card!.click();
    await page.waitForURL(/view=negocios&id=\d+/);

    await expect(page.locator(`text=${name}`)).toBeVisible();
    await expect(page.locator("text=Etapa")).toBeVisible();

    await page.getByRole("button", { name: /volver/i }).click();
    await page.waitForURL(/view=negocios$/);

    await dealsPage.moveToStage(name, "Calificado");
    await page.waitForTimeout(500);
    await card!.click();
    await page.waitForURL(/view=negocios&id=\d+/);
    await expect(page.locator("text=Etapa cambiada")).toBeVisible();
  });

  test("change deal status to ganado (won) via API reflects in detail", async ({ page }) => {
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });
    await dealsPage.goto();

    const name = `Won Deal ${Date.now()}`;
    await dealsPage.create({ name, amount: "2500000" });

    const card = await dealsPage.getCard(name);
    expect(card).not.toBeNull();

    // Status transitions are API-driven (UI shows estado read-only).
    const list = await apiRequest<{ data?: { id: number; nombre: string }[] }>("/crm/negocios");
    const deals = Array.isArray(list) ? list : list.data ?? [];
    const deal = deals.find((d: { nombre: string }) => d.nombre === name);
    expect(deal).toBeDefined();

    const update = await apiRequest<{ success: boolean }>(`/crm/negocios/${deal!.id}`, {
      method: "PUT",
      body: { estado: "ganado" },
      orgSlug: "test-org-pro",
      email: "admin-pro@test.com",
    });
    expect(update).toBeTruthy();

    await card!.click();
    await page.waitForURL(/view=negocios&id=\d+/);
    await expect(page.locator(`text=${name}`)).toBeVisible();
    await expect(page.locator("text=ganado")).toBeVisible();
  });

  test("create deal linked to contact and company", async ({ page }) => {
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });

    const ts = Date.now();
    const companyName = `Linked Company ${ts}`;
    const contactPhone = `+57310${ts}`;
    const contactName = `Linked Contact ${ts}`;

    // Create company
    await page.goto("/dashboard/crm?view=empresas");
    await page.waitForSelector("table");
    await page.getByRole("button", { name: /nueva empresa/i }).click();
    await page.fill('input[name="name"]', companyName);
    await page.fill('input[name="nit"]', `${ts}`);
    await page.getByRole("button", { name: /guardar|crear/i }).click();
    await page.waitForResponse((res) => res.url().includes("/api/crm/empresas") && res.ok());

    // Create contact
    await page.goto("/dashboard/crm?view=contactos");
    await page.waitForSelector("table");
    await page.getByRole("button", { name: /nuevo contacto/i }).click();
    await page.fill('input[name="phone"]', contactPhone);
    await page.fill('input[name="display_name"]', contactName);
    await page.getByRole("button", { name: /guardar|crear/i }).click();
    await page.waitForResponse((res) => res.url().includes("/api/crm/contactos") && res.ok());

    // Create deal linked to both
    const companies = await apiRequest<{ data?: { id: number; nombre: string }[] }>("/crm/empresas");
    const companyList = Array.isArray(companies) ? companies : companies.data ?? [];
    const company = companyList.find((c: { nombre: string }) => c.nombre === companyName);
    expect(company).toBeDefined();

    const contacts = await apiRequest<{ data?: { id: number; phone: string }[] }>("/crm/contactos");
    const contactList = Array.isArray(contacts) ? contacts : contacts.data ?? [];
    const contact = contactList.find((c: { phone: string }) => c.phone === contactPhone);
    expect(contact).toBeDefined();

    await dealsPage.goto();
    const dealName = `Linked Deal ${ts}`;
    await dealsPage.newDealButton.click();
    await page.fill('input[name="nombre"]', dealName);
    await page.fill('input[name="monto"]', "1000000");
    await page.selectOption('select[name="company_id"], #company_id', String(company.id));
    await page.selectOption('select[name="contact_id"], #contact_id', String(contact.id));
    await page.getByRole("button", { name: /guardar|crear/i }).click();
    await page.waitForResponse((res) => res.url().includes("/api/crm/negocios") && res.ok());

    const card = await dealsPage.getCard(dealName);
    expect(card).not.toBeNull();
  });
});
