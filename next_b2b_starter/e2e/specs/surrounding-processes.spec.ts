import { test, expect } from "@playwright/test";
import { apiRequest } from "../helpers/api";
import { uniqueColombianPhone } from "../helpers/phones";
import {
  buildWhatsAppTextPayload,
  deliverWebhook,
  seedWhatsAppConfig,
} from "../helpers/whatsapp";

const PHONE_NUMBER_ID = "111222333";
const WEBHOOK_SECRET = "test_webhook_secret_for_e2e";

interface ContactDto {
  id: number;
  phone_number?: string;
  display_name?: string;
}

interface ConversationDto {
  id: number;
  contact_phone?: string;
  contact_display_name?: string;
}

interface MessageDto {
  direction?: string;
  content?: string;
}

async function listContacts(orgSlug: string, email: string): Promise<ContactDto[]> {
  const res = await apiRequest<{ data?: ContactDto[] }>("/crm/contactos", { orgSlug, email });
  return Array.isArray(res) ? res : res.data ?? [];
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

test.describe("Surrounding processes", () => {
  test.beforeAll(async () => {
    await seedWhatsAppConfig({
      phoneNumberId: PHONE_NUMBER_ID,
      webhookSecret: WEBHOOK_SECRET,
      verifyToken: "test_verify_token_for_e2e",
    });
  });

  test("cross-org isolation: pro-created contact absent from free org list", async () => {
    const phone = uniqueColombianPhone();
    const name = `Isolation ${Date.now()}`;

    await apiRequest("/crm/contactos", {
      method: "POST",
      body: { phone_number: phone, display_name: name },
      orgSlug: "test-org-pro",
      email: "admin-pro@test.com",
    });

    const freeContacts = await listContacts("test-org-free", "admin-free@test.com");
    expect(freeContacts.some((c) => c.phone_number === phone)).toBe(false);

    const proContacts = await listContacts("test-org-pro", "admin-pro@test.com");
    expect(proContacts.some((c) => c.phone_number === phone)).toBe(true);
  });

  test("pagination: default limit returns 20, explicit limit/offset retrieves remainder", async () => {
    const seeded: string[] = [];
    for (let i = 0; i < 25; i++) {
      const phone = uniqueColombianPhone();
      await apiRequest("/crm/contactos", {
        method: "POST",
        body: { phone_number: phone, display_name: `Page ${Date.now()} ${i}` },
        orgSlug: "test-org-pro",
        email: "admin-pro@test.com",
      });
      seeded.push(phone);
    }

    const defaultPage = await listContacts("test-org-pro", "admin-pro@test.com");
    expect(defaultPage.length).toBe(20);

    const res = await apiRequest<{ data?: ContactDto[] }>(
      "/crm/contactos?limit=25&offset=20",
      { orgSlug: "test-org-pro", email: "admin-pro@test.com" }
    );
    const page2 = Array.isArray(res) ? res : res.data ?? [];
    expect(page2.length).toBeGreaterThan(0);
    expect(defaultPage.some((c) => c.phone_number === page2[0].phone_number)).toBe(false);
  });

  test("reply persists as an outbound message", async () => {
    const from = `+57315000${Date.now() % 10000}`;
    const body = `Mensaje entrante ${Date.now()}`;
    const payload = buildWhatsAppTextPayload({ phoneNumberId: PHONE_NUMBER_ID, from, body });
    const deliverRes = await deliverWebhook(WEBHOOK_SECRET, payload);
    expect(deliverRes.status).toBe(200);

    const conv = await findConversationByPhone(from);
    expect(conv).toBeDefined();

    const reply = `Respuesta ${Date.now()}`;
    const sentRes = await apiRequest<{ data?: MessageDto }>(`/crm/conversaciones/${conv!.id}/mensajes`, {
      method: "POST",
      body: { content: reply },
      orgSlug: "test-org-pro",
      email: "admin-pro@test.com",
    });
    const sent = (sentRes.data ?? sentRes) as MessageDto;
    expect(sent.direction).toBe("outbound");

    const msgs = await apiRequest<{ data?: MessageDto[] }>(
      `/crm/conversaciones/${conv!.id}/mensajes`
    );
    const list = Array.isArray(msgs) ? msgs : msgs.data ?? [];
    const outbound = list.find((m) => m.direction === "outbound" && m.content === reply);
    expect(outbound).toBeDefined();
  });

  test("mock-auth guard: request without X-Test-Org-ID returns 401", async () => {
    const res = await fetch("http://localhost:8080/api/crm/contactos");
    expect(res.status).toBe(401);
  });

  test("member cannot access org:manage-gated WhatsApp config (403)", async () => {
    const res = await fetch("http://localhost:8080/api/v1/whatsapp/config", {
      method: "GET",
      headers: { "X-Test-Org-ID": "test-org-pro:member-pro@test.com" },
    });
    expect(res.status).toBe(403);
  });
});
