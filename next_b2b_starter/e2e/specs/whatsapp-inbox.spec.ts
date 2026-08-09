import { test, expect } from "@playwright/test";
import {
  buildWhatsAppTextPayload,
  signWebhookBody,
  deliverWebhook,
  verifyChallenge,
  seedWhatsAppConfig,
} from "../helpers/whatsapp";
import { apiRequest } from "../helpers/api";
import { InboxPage } from "../page-objects/inbox.page";

const PHONE_NUMBER_ID = "111222333";
const WEBHOOK_SECRET = "test_webhook_secret_for_e2e";
const VERIFY_TOKEN = "test_verify_token_for_e2e";

interface ConversationDto {
  id: number;
  contact_phone?: string;
  contact_display_name?: string;
  status?: string;
}

interface MessageDto {
  id: number;
  whatsapp_message_id?: string;
  content?: string;
  direction?: string;
}

async function findConversationByPhone(phone: string): Promise<ConversationDto | undefined> {
  const res = await apiRequest<{ data?: ConversationDto[] }>("/crm/conversaciones");
  const list = Array.isArray(res) ? res : res.data ?? [];
  return list.find((c) => c.contact_phone === phone || c.contact_display_name === phone);
}

test.describe("WhatsApp Inbox", () => {
  test.beforeAll(async () => {
    await seedWhatsAppConfig({
      phoneNumberId: PHONE_NUMBER_ID,
      webhookSecret: WEBHOOK_SECRET,
      verifyToken: VERIFY_TOKEN,
    });
  });

  test("signed inbound text webhook renders conversation + message in inbox thread", async ({
    page,
  }) => {
    const from = `+57314111${Date.now() % 10000}`;
    const body = `Hola, quiero información del plan ${Date.now()}`;
    const payload = buildWhatsAppTextPayload({ phoneNumberId: PHONE_NUMBER_ID, from, body });

    const res = await deliverWebhook(WEBHOOK_SECRET, payload);
    expect(res.status).toBe(200);

    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });
    const inbox = new InboxPage(page);
    await inbox.goto();
    const conv = await inbox.getConversation(from);
    expect(conv).not.toBeNull();

    await inbox.openConversation(from);
    await expect(page.locator(`[data-testid="message-thread"] :text("${body}")`).first()).toBeVisible();
  });

  test("duplicate delivery persists exactly one message (idempotency)", async () => {
    const from = `+57314222${Date.now() % 10000}`;
    const messageId = `wamid.dup.${Date.now()}`;
    const body = `Mensaje duplicado ${Date.now()}`;
    const payload = buildWhatsAppTextPayload({
      phoneNumberId: PHONE_NUMBER_ID,
      from,
      body,
      messageId,
    });

    for (let i = 0; i < 2; i++) {
      const res = await deliverWebhook(WEBHOOK_SECRET, payload);
      expect(res.status).toBe(200);
    }

    const conv = await findConversationByPhone(from);
    expect(conv).toBeDefined();

    const msgs = await apiRequest<{ data?: MessageDto[] }>(`/crm/conversaciones/${conv!.id}/mensajes`);
    const list = Array.isArray(msgs) ? msgs : msgs.data ?? [];
    const dup = list.filter((m) => m.whatsapp_message_id === messageId);
    expect(dup.length).toBe(1);
  });

  test("invalid HMAC returns 401 and no message appears in inbox", async ({ page }) => {
    const from = `+57314333${Date.now() % 10000}`;
    const body = `Mensaje con firma inválida ${Date.now()}`;
    const payload = buildWhatsAppTextPayload({ phoneNumberId: PHONE_NUMBER_ID, from, body });

    const res = await deliverWebhook(WEBHOOK_SECRET, payload, { tamperSignature: true });
    expect(res.status).toBe(401);

    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });
    const inbox = new InboxPage(page);
    await inbox.goto();
    const conv = await inbox.getConversation(from);
    expect(conv).toBeNull();
  });

  test("unknown phone_number_id returns 404", async () => {
    const from = `+57314444${Date.now() % 10000}`;
    const payload = buildWhatsAppTextPayload({
      phoneNumberId: "9999999999",
      from,
      body: "Number no configurado",
    });

    const res = await deliverWebhook(WEBHOOK_SECRET, payload);
    expect(res.status).toBe(404);
  });

  test("verification handshake returns the challenge", async () => {
    const challenge = `challenge_${Date.now()}`;
    const res = await verifyChallenge(VERIFY_TOKEN, challenge);
    expect(res.status).toBe(200);
    expect(res.body).toContain(challenge);
  });

  test("webhook_logs reflected in config health stats", async () => {
    const from = `+57314555${Date.now() % 10000}`;
    const payload = buildWhatsAppTextPayload({
      phoneNumberId: PHONE_NUMBER_ID,
      from,
      body: `Mensaje para health ${Date.now()}`,
    });
    const res = await deliverWebhook(WEBHOOK_SECRET, payload);
    expect(res.status).toBe(200);

    const health = await apiRequest<{ total?: number }>("/v1/whatsapp/config/health");
    expect(health).toHaveProperty("total");
    expect(typeof health.total).toBe("number");
  });
});
