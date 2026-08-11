import { test, expect } from "@playwright/test";
import {
  buildWhatsAppTextPayload,
  buildEchoTextPayload,
  signWebhookBody,
  deliverWebhook,
  verifyChallenge,
  seedWhatsAppConfig,
  setConfigActive,
} from "../helpers/whatsapp";
import { apiRequest } from "../helpers/api";

const PHONE_NUMBER_ID = "111222333";
const WEBHOOK_SECRET = "test_webhook_secret_for_e2e";
const VERIFY_TOKEN = "test_verify_token_for_e2e";
const WRONG_TOKEN = "wrong_verify_token_for_e2e";

interface ConversationDto {
  id: number;
  contact_phone?: string;
  contact_display_name?: string;
}

interface MessageDto {
  provider_message_id?: string;
  direction?: string;
}

async function findConversationByPhone(phone: string): Promise<ConversationDto | undefined> {
  for (let attempt = 0; attempt < 10; attempt++) {
    const res = await apiRequest<{ data?: ConversationDto[] }>("/crm/conversaciones");
    const list = Array.isArray(res) ? res : res.data ?? [];
    const found = list.find((c) => c.contact_phone === phone || c.contact_display_name === phone);
    if (found) return found;
    await new Promise((r) => setTimeout(r, 500));
  }
  return undefined;
}

async function findMessageByProviderId(convId: number, messageId: string): Promise<MessageDto | undefined> {
  for (let attempt = 0; attempt < 10; attempt++) {
    const res = await apiRequest<{ data?: MessageDto[] }>(`/crm/conversaciones/${convId}/mensajes`);
    const list = Array.isArray(res) ? res : res.data ?? [];
    const found = list.find((m) => m.provider_message_id === messageId);
    if (found) return found;
    await new Promise((r) => setTimeout(r, 500));
  }
  return undefined;
}

test.describe("WhatsApp webhook edge cases", () => {
  test.beforeAll(async () => {
    await seedWhatsAppConfig({
      phoneNumberId: PHONE_NUMBER_ID,
      webhookSecret: WEBHOOK_SECRET,
      verifyToken: VERIFY_TOKEN,
    });
  });

  test("inactive config returns 404 unknown_phone_number", async () => {
    await setConfigActive({ phoneNumberId: PHONE_NUMBER_ID, isActive: false });

    try {
      const from = `+57314666${Date.now() % 10000}`;
      const payload = buildWhatsAppTextPayload({
        phoneNumberId: PHONE_NUMBER_ID,
        from,
        body: "Mensaje con config inactiva",
      });
      const res = await deliverWebhook(WEBHOOK_SECRET, payload);
      expect(res.status).toBe(404);
      expect((res.body as { code?: string }).code).toBe("unknown_phone_number");
    } finally {
      await setConfigActive({ phoneNumberId: PHONE_NUMBER_ID, isActive: true });
    }
  });

  test("invalid verify_token handshake returns 403", async () => {
    const res = await verifyChallenge(WRONG_TOKEN, "challenge_should_fail");
    expect(res.status).toBe(403);
  });

  test("malformed JSON payload returns 400 invalid_json", async () => {
    const signature = signWebhookBody(WEBHOOK_SECRET, "this is not json");
    const res = await fetch("http://localhost:8080/api/v1/webhooks/whatsapp", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "x-hub-signature-256": signature,
      },
      body: "this is not json",
    });
    expect(res.status).toBe(400);
    const body = await res.json();
    expect(body.code).toBe("invalid_json");
  });

  test("failed webhook is logged in health stats", async () => {
    const from = `+57314777${Date.now() % 10000}`;
    const payload = buildWhatsAppTextPayload({
      phoneNumberId: PHONE_NUMBER_ID,
      from,
      body: "Mensaje con firma inválida",
    });

    const res = await deliverWebhook(WEBHOOK_SECRET, payload, { tamperSignature: true });
    expect(res.status).toBe(401);

    const health = await apiRequest<{ by_status?: Record<string, number> }>("/v1/whatsapp/config/health");
    expect(health.by_status?.failed ?? 0).toBeGreaterThan(0);
  });

  test("inbound message carries direction=inbound", async () => {
    const from = `+57314888${Date.now() % 10000}`;
    const messageId = `wamid.dir.${Date.now()}`;
    const body = `Mensaje con dirección ${Date.now()}`;
    const payload = buildWhatsAppTextPayload({
      phoneNumberId: PHONE_NUMBER_ID,
      from,
      body,
      messageId,
    });

    const res = await deliverWebhook(WEBHOOK_SECRET, payload);
    expect(res.status).toBe(200);

    const conv = await findConversationByPhone(from);
    expect(conv).toBeDefined();
    const msg = await findMessageByProviderId(conv!.id, messageId);
    expect(msg).toBeDefined();
    expect(msg!.direction).toBe("inbound");
  });

  test("echo message is not persisted as inbound", async () => {
    const from = `+57314999${Date.now() % 10000}`;
    const messageId = `wamid.echo.test.${Date.now()}`;
    const body = "Echo desde la app";
    const payload = buildEchoTextPayload({
      phoneNumberId: PHONE_NUMBER_ID,
      from,
      body,
      messageId,
    });

    const res = await deliverWebhook(WEBHOOK_SECRET, payload);
    expect(res.status).toBe(200);

    const conv = await findConversationByPhone(from);
    if (conv) {
      const msg = await findMessageByProviderId(conv.id, messageId);
      expect(msg).toBeDefined();
      expect(msg!.direction).not.toBe("inbound");
    }
  });
});
