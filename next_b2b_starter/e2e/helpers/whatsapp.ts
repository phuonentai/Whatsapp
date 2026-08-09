import crypto from "crypto";
import { apiRequest } from "./api";

const WEBHOOK_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

/**
 * Build a WhatsApp Cloud API webhook payload for one inbound text message.
 * Shape mirrors the real Meta Cloud API: entry[].changes[].value.{metadata,messages}.
 */
export function buildWhatsAppTextPayload(opts: {
  phoneNumberId: string;
  from: string;
  body: string;
  messageId?: string;
  timestamp?: string;
}): string {
  const messageId = opts.messageId ?? `wamid.${Date.now()}.${Math.random().toString(36).slice(2, 10)}`;
  const timestamp = opts.timestamp ?? String(Math.floor(Date.now() / 1000));
  const payload = {
    object: "whatsapp_business_account",
    entry: [
      {
        id: "0",
        changes: [
          {
            field: "messages",
            value: {
              messaging_product: "whatsapp",
              metadata: {
                display_phone_number: "15550123456",
                phone_number_id: opts.phoneNumberId,
              },
              contacts: [{ profile: { name: "Test Contact" }, wa_id: opts.from }],
              messages: [
                {
                  from: opts.from,
                  id: messageId,
                  timestamp,
                  text: { body: opts.body },
                  type: "text",
                },
              ],
            },
          },
        ],
      },
    ],
  };
  return JSON.stringify(payload);
}

/**
 * HMAC-SHA256 sign a raw webhook body with the org's webhook secret,
 * producing the `x-hub-signature-256: sha256=<hex>` header value.
 */
export function signWebhookBody(secret: string, body: string): string {
  const rawHMAC = crypto.createHmac("sha256", secret);
  rawHMAC.update(body);
  return `sha256=${rawHMAC.digest("hex")}`;
}

/**
 * POST a signed webhook payload to the real Go ingress endpoint. Returns the
 * HTTP status and response body so tests can assert success/error paths.
 */
export async function deliverWebhook(
  secret: string,
  body: string,
  opts: { phoneNumberId?: string; tamperSignature?: boolean } = {}
): Promise<{ status: number; body: unknown }> {
  const signature = opts.tamperSignature ? "sha256=deadbeef" : signWebhookBody(secret, body);
  const res = await fetch(`${WEBHOOK_URL}/webhooks/whatsapp`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "x-hub-signature-256": signature,
      ...(opts.phoneNumberId ? { "X-Test-Phone-Number-ID": opts.phoneNumberId } : {}),
    },
    body,
  });
  let json: unknown = null;
  try {
    json = await res.json();
  } catch {
    // empty body is expected for 200 OK
  }
  return { status: res.status, body: json };
}

/**
 * Verify handshake (hub.mode=subscribe) with a GET request.
 */
export async function verifyChallenge(
  verifyToken: string,
  challenge: string
): Promise<{ status: number; body: string }> {
  const res = await fetch(
    `${WEBHOOK_URL}/webhooks/whatsapp?hub.mode=subscribe&hub.verify_token=${verifyToken}&hub.challenge=${challenge}`
  );
  return { status: res.status, body: await res.text() };
}

/**
 * Seed (upsert) the WhatsApp config for the pro test org via the management API
 * under mock auth, returning the known webhook_secret / verify_token for HMAC.
 */
export async function seedWhatsAppConfig(opts: {
  phoneNumberId: string;
  webhookSecret: string;
  verifyToken: string;
  accessToken?: string;
}): Promise<void> {
  await apiRequest("/v1/whatsapp/config", {
    method: "PUT",
    body: {
      phone_number_id: opts.phoneNumberId,
      webhook_secret: opts.webhookSecret,
      verify_token: opts.verifyToken,
      access_token: opts.accessToken ?? "TEST_ACCESS_TOKEN",
      is_active: true,
    },
    orgSlug: "test-org-pro",
    email: "admin-pro@test.com",
  });
}
