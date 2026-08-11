import { test, expect } from "@playwright/test";
import { buildWhatsAppTextPayload, deliverWebhook, seedWhatsAppConfig } from "../helpers/whatsapp";
import { apiRequest } from "../helpers/api";
import { InboxPage } from "../page-objects/inbox.page";
import { uniqueColombianPhone } from "../helpers/phones";

const PHONE_NUMBER_ID = "222333444";
const WEBHOOK_SECRET = "inbox_ui_secret_e2e";
const VERIFY_TOKEN = "inbox_ui_verify_token";
const ORG = { orgSlug: "test-org-pro", email: "admin-pro@test.com" };

interface ConversationDto {
  id: number;
  contact_phone?: string;
  contact_display_name?: string;
  status?: string;
}

async function createConversation(prefix: string): Promise<{ phone: string; body: string; conv: ConversationDto }> {
  const phone = uniqueColombianPhone();
  const body = `Mensaje inbox-ui ${Date.now()}`;
  const payload = buildWhatsAppTextPayload({ phoneNumberId: PHONE_NUMBER_ID, from: phone, body });
  const res = await deliverWebhook(WEBHOOK_SECRET, payload);
  expect(res.status).toBe(200);

  const conv = await findConversationByPhone(phone);
  expect(conv).toBeDefined();
  return { phone, body, conv: conv! };
}

async function findConversationByPhone(phone: string): Promise<ConversationDto | undefined> {
  for (let attempt = 0; attempt < 10; attempt++) {
    const res = await apiRequest<{ data?: ConversationDto[] }>("/crm/conversaciones", ORG);
    const list = Array.isArray(res) ? res : res.data ?? [];
    const found = list.find((c) => c.contact_phone === phone || c.contact_display_name === phone);
    if (found) return found;
    await new Promise((r) => setTimeout(r, 500));
  }
  return undefined;
}

test.describe("Inbox UI", () => {
  test.beforeAll(async () => {
    await seedWhatsAppConfig({
      phoneNumberId: PHONE_NUMBER_ID,
      webhookSecret: WEBHOOK_SECRET,
      verifyToken: VERIFY_TOKEN,
    });
  });

  test("typed reply appears in the thread and input clears", async ({ page }) => {
    const { phone, body } = await createConversation("reply");
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });
    const inbox = new InboxPage(page);
    await inbox.goto();
    await inbox.openConversation(phone);
    await expect(page.locator(`[data-testid="message-thread"] :text("${body}")`).first()).toBeVisible();

    const reply = `Respuesta de prueba ${Date.now()}`;
    // Outbound send hits the real WhatsApp Graph API; mock the message
    // endpoints so the reply round-trips deterministically in the UI.
    await page.route("**/api/crm/conversaciones/*/mensajes", async (route) => {
      const method = route.request().method();
      if (method === "POST") {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({
            success: true,
            data: {
              id: Date.now(),
              organization_id: 2,
              conversation_id: 1,
              contact_id: 1,
              provider_message_id: "mock-outbound",
              direction: "outbound",
              message_type: "text",
              content: reply,
              status: "sent",
              created_at: new Date().toISOString(),
              updated_at: new Date().toISOString(),
            },
          }),
        });
      } else {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({
            success: true,
            data: [
              {
                id: Date.now(),
                organization_id: 2,
                conversation_id: 1,
                contact_id: 1,
                provider_message_id: "mock-inbound",
                direction: "inbound",
                message_type: "text",
                content: body,
                status: "received",
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
              },
              {
                id: Date.now() + 1,
                organization_id: 2,
                conversation_id: 1,
                contact_id: 1,
                provider_message_id: "mock-outbound",
                direction: "outbound",
                message_type: "text",
                content: reply,
                status: "sent",
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
              },
            ],
          }),
        });
      }
    });

    await inbox.sendReply(reply);
    await expect(page.getByPlaceholder("Type a message...")).toHaveValue("");
  });

  test("empty reply is not sent", async ({ page }) => {
    const { phone, body } = await createConversation("empty");
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });
    const inbox = new InboxPage(page);
    await inbox.goto();
    await inbox.openConversation(phone);
    // Wait for the inbound message to render before snapshotting the thread so
    // the before/after comparison is stable against async message loading.
    await expect(page.locator(`[data-testid="message-thread"] :text("${body}")`).first()).toBeVisible();

    const { before, after } = await inbox.sendEmptyReply();
    expect(after).toBe(before);
  });

  test("status filter narrows the conversation list", async ({ page }) => {
    const { phone, conv } = await createConversation("status");
    await apiRequest(`/crm/conversaciones/${conv.id}/status`, {
      method: "PATCH",
      body: { status: "closed" },
      ...ORG,
    });

    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });
    const inbox = new InboxPage(page);
    await inbox.goto();

    await inbox.setStatusFilter("Closed");
    await inbox.assertConversationStatus(phone, "closed");

    await inbox.setStatusFilter("Active");
    const activeRow = await inbox.getConversation(phone);
    expect(activeRow).toBeNull();
  });

  test("quick reply pill fills the reply input from an applied playbook", async ({ page }) => {
    await apiRequest("/playbooks/comercio/apply", { method: "POST", body: {}, ...ORG });

    const { phone } = await createConversation("quick");
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });
    const inbox = new InboxPage(page);
    await inbox.goto();
    await inbox.openConversation(phone);

    await inbox.selectQuickReply("Saludo", "¡Hola! Gracias por escribirnos. ¿En qué podemos ayudarte hoy?");
  });

  test("no applied playbooks hides the quick-replies row", async ({ page }) => {
    await apiRequest("/playbooks/comercio/reset", { method: "POST", body: {}, ...ORG });

    const { phone } = await createConversation("noquick");
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });
    const inbox = new InboxPage(page);
    await inbox.goto();
    await inbox.openConversation(phone);

    expect(await inbox.quickRepliesVisible()).toBe(false);
  });

  test("scripted sequence auto-advances the composer after each send", async ({ page }) => {
    await apiRequest("/playbooks/comercio/apply", { method: "POST", body: {}, ...ORG });

    const { phone } = await createConversation("seq");
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });

    // Outbound send hits the real WhatsApp Graph API; mock the message
    // endpoints so each step round-trips deterministically.
    await page.route("**/api/crm/conversaciones/*/mensajes", async (route) => {
      if (route.request().method() === "POST") {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({
            success: true,
            data: { id: Date.now(), direction: "outbound", content: "ok", status: "sent" },
          }),
        });
      } else {
        await route.continue();
      }
    });

    const inbox = new InboxPage(page);
    await inbox.goto();
    await inbox.openConversation(phone);

    const input = page.getByPlaceholder("Type a message...");
    const stepOne = "¡Perfecto! ¿Qué producto(s) quieres y en qué cantidad?";
    const stepTwo = "¿A qué dirección lo enviamos? ¿Algún punto de referencia?";
    const stepThree = "Te enviamos el link de pago: puedes pagar con PSE, Nequi o tarjeta. Cuando esté confirmado, lo despachamos.";

    await page.getByRole("button", { name: /Confirmar pedido/ }).click();
    await expect(input).toHaveValue(stepOne);
    await expect(page.getByText("Paso 1 de 3")).toBeVisible();

    await input.press("Enter");
    await expect(input).toHaveValue(stepTwo);
    await expect(page.getByText("Paso 2 de 3")).toBeVisible();

    await input.press("Enter");
    await expect(input).toHaveValue(stepThree);
    await expect(page.getByText("Paso 3 de 3")).toBeVisible();

    await input.press("Enter");
    await expect(page.getByText(/Paso \d de \d/)).not.toBeVisible();
    await expect(input).toHaveValue("");
  });

  test("failed sequence send does not advance", async ({ page }) => {
    await apiRequest("/playbooks/comercio/apply", { method: "POST", body: {}, ...ORG });

    const { phone } = await createConversation("seqfail");
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });
    const inbox = new InboxPage(page);
    await inbox.goto();
    await inbox.openConversation(phone);

    await page.route("**/api/crm/conversaciones/*/mensajes", async (route) => {
      if (route.request().method() === "POST") {
        await route.fulfill({
          status: 500,
          contentType: "application/json",
          body: JSON.stringify({ error: "boom" }),
        });
      } else {
        await route.continue();
      }
    });

    const input = page.getByPlaceholder("Type a message...");
    const stepOne = "¡Perfecto! ¿Qué producto(s) quieres y en qué cantidad?";

    await page.getByRole("button", { name: /Confirmar pedido/ }).click();
    await expect(input).toHaveValue(stepOne);
    await expect(page.getByText("Paso 1 de 3")).toBeVisible();

    await input.press("Enter");
    await page.waitForResponse((res) => res.url().includes("/mensajes") && res.status() === 500);
    await expect(input).toHaveValue(stepOne);
    await expect(page.getByText("Paso 1 de 3")).toBeVisible();
  });

  test("switching conversation resets sequence mode", async ({ page }) => {
    await apiRequest("/playbooks/comercio/apply", { method: "POST", body: {}, ...ORG });

    const first = await createConversation("seqswitch");
    const second = await createConversation("seqswitch2");
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });

    await page.route("**/api/crm/conversaciones/*/mensajes", async (route) => {
      if (route.request().method() === "POST") {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({
            success: true,
            data: { id: Date.now(), direction: "outbound", content: "ok", status: "sent" },
          }),
        });
      } else {
        await route.continue();
      }
    });

    const inbox = new InboxPage(page);
    await inbox.goto();
    await inbox.openConversation(first.phone);

    const input = page.getByPlaceholder("Type a message...");
    const stepOne = "¡Perfecto! ¿Qué producto(s) quieres y en qué cantidad?";

    await page.getByRole("button", { name: /Confirmar pedido/ }).click();
    await expect(input).toHaveValue(stepOne);
    await expect(page.getByText("Paso 1 de 3")).toBeVisible();

    await inbox.openConversation(second.phone);
    await expect(page.getByText(/Paso \d de \d/)).not.toBeVisible();
    await expect(input).toHaveValue("");
  });

  test("approving a pending suggestion removes it from the panel", async ({ page }) => {
    const { phone, conv } = await createConversation("approve");
    await seedSuggestion(conv.id, `Borrador aprobable ${Date.now()}`);
    let resolved = false;

    // Mock both the suggestions list and the approve call so the panel
    // transitions deterministically (approve would otherwise hit Meta outbound).
    await page.route("**/api/agent/suggestions?status=pending", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          suggestions: resolved
            ? []
            : [
                {
                  id: 999,
                  conversation_id: conv.id,
                  type: "reply",
                  body: "Borrador aprobable",
                  status: "pending",
                  source: "copilot",
                },
              ],
        }),
      })
    );
    await page.route("**/api/agent/suggestions/*/approve", (route) => {
      resolved = true;
      return route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          success: true,
          data: {
            id: 999,
            conversation_id: conv.id,
            type: "reply",
            body: "mock",
            status: "approved",
            source: "copilot",
          },
        }),
      });
    });

    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });
    const inbox = new InboxPage(page);
    await inbox.goto();
    await inbox.openConversation(phone);

    await expect(page.getByRole("button", { name: "Aprobar y enviar" })).toBeVisible();
    await inbox.approveSuggestion();
  });
  test("rejecting a pending suggestion dismisses it from the panel", async ({ page }) => {
    const { phone, conv } = await createConversation("reject");
    await seedSuggestion(conv.id, `Borrador rechazable ${Date.now()}`);
    let resolved = false;

    await page.route("**/api/agent/suggestions?status=pending", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          suggestions: resolved
            ? []
            : [
                {
                  id: 998,
                  conversation_id: conv.id,
                  type: "reply",
                  body: "Borrador rechazable",
                  status: "pending",
                  source: "copilot",
                },
              ],
        }),
      })
    );
    await page.route("**/api/agent/suggestions/*/reject", (route) => {
      resolved = true;
      return route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          success: true,
          data: {
            id: 998,
            conversation_id: conv.id,
            type: "reply",
            body: "mock",
            status: "rejected",
            source: "copilot",
          },
        }),
      });
    });

    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });
    const inbox = new InboxPage(page);
    await inbox.goto();
    await inbox.openConversation(phone);

    await expect(page.getByRole("button", { name: "Rechazar" })).toBeVisible();
    await inbox.rejectSuggestion();
  });

  test("member identity is redirected away from the inbox", async ({ page }) => {
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:member-pro@test.com" });
    await page.goto("/dashboard/inbox");
    await page.waitForURL((url) => !url.pathname.includes("/inbox"));
    expect(page.url()).toContain("/dashboard");
  });

  test("member approve-suggestion API returns 403", async ({ page }) => {
    const res = await page.request.post("http://localhost:3001/api/agent/suggestions/999999/approve", {
      data: {},
      headers: { "X-Test-Org-ID": "test-org-pro:member-pro@test.com" },
    });
    expect(res.status()).toBe(403);
  });

  test("long reply sends intact", async ({ page }) => {
    const { phone } = await createConversation("long");
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });
    const inbox = new InboxPage(page);
    await inbox.goto();
    await inbox.openConversation(phone);

    const longText = `L${"a".repeat(2000)}Z ${Date.now()}`;
    await inbox.sendReply(longText);
    await expect(page.locator(`[data-testid="message-thread"] :text("${longText}")`).first()).toBeVisible();
  });

  test("unicode reply round-trips in the thread", async ({ page }) => {
    const { phone } = await createConversation("unicode");
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });
    const inbox = new InboxPage(page);
    await inbox.goto();
    await inbox.openConversation(phone);

    const unicode = `Hola 👋 ñoño ünïcode ${Date.now()}`;
    await inbox.sendReply(unicode);
    await expect(page.locator(`[data-testid="message-thread"] :text("${unicode}")`).first()).toBeVisible();
  });

  test("failed reply is not appended to the thread", async ({ page }) => {
    const { phone } = await createConversation("fail");
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });
    const inbox = new InboxPage(page);
    await inbox.goto();
    await inbox.openConversation(phone);

    await page.route("**/api/crm/conversaciones/*/mensajes", async (route) => {
      if (route.request().method() === "POST") {
        await route.fulfill({
          status: 500,
          contentType: "application/json",
          body: JSON.stringify({ error: "boom" }),
        });
      } else {
        await route.continue();
      }
    });

    const text = `Fallo ${Date.now()}`;
    await page.getByPlaceholder("Type a message...").fill(text);
    await page.getByPlaceholder("Type a message...").press("Enter");
    await page.waitForResponse((res) => res.url().includes("/mensajes") && res.status() === 500);
    await expect(page.locator(`[data-testid="message-thread"] :text("${text}")`).first()).not.toBeVisible();
  });
});

async function seedSuggestion(conversationId: number, body: string): Promise<void> {
  await apiRequest("/agent/suggestions/seed", {
    method: "POST",
    body: { conversation_id: conversationId, body },
    ...ORG,
  });
}
