import { test, expect } from "@playwright/test";
import path from "path";
import fs from "fs";
import { buildWhatsAppTextPayload, deliverWebhook, seedWhatsAppConfig } from "../helpers/whatsapp";
import { uniqueColombianPhone } from "../helpers/phones";
import { apiRequest } from "../helpers/api";

const PHONE_NUMBER_ID = "222333445";
const WEBHOOK_SECRET = "inbox_member_qa_secret";
const MEMBER_ORG = { orgSlug: "test-org-rbac", email: "member-rbac@test.com" };

// Visual + a11y QA for inbox-member-tier (member vs admin tiers).
// Artifacts → openspec/changes/inbox-member-tier/qa/
const QA_DIR = path.resolve(
  __dirname,
  "../../../openspec/changes/inbox-member-tier/qa"
);
fs.mkdirSync(QA_DIR, { recursive: true });

const VIEWPORTS = [
  { name: "390x844", width: 390, height: 844 },
  { name: "768x1024", width: 768, height: 1024 },
  { name: "1440x900", width: 1440, height: 900 },
];

const ADMIN = "test-org-pro:admin-pro@test.com";
const MEMBER = "test-org-rbac:member-rbac@test.com";

const report: Array<Record<string, unknown>> = [];

test.describe.configure({ mode: "serial" });

test("seed conversations for QA", async ({ page, request }) => {
  // Admin org: reuse the most recent conversation.
  const list = await request.get("http://localhost:8080/api/crm/conversaciones?limit=1", {
    headers: { "X-Test-Org-ID": ADMIN },
  });
  const body = await list.json();
  const conv = (body.data ?? body.conversations ?? [])[0];
  if (conv) {
    process.env.QA_CONV_PHONE_ADMIN = String(conv.contact_phone ?? conv.contactPhone ?? "");
  }

  // Member org: ensure it HAS a conversation so the member tier (read +
  // manual reply) is exercisable. Seed its own WhatsApp config and deliver an
  // inbound webhook for a unique phone.
  const memberPhone = uniqueColombianPhone();
  await seedWhatsAppConfig({
    phoneNumberId: PHONE_NUMBER_ID,
    webhookSecret: WEBHOOK_SECRET,
    verifyToken: "verify_member_qa",
  });
  // The member org config lives under test-org-rbac (admin-rbac can manage it).
  await apiRequest("/v1/whatsapp/config", {
    method: "PUT",
    body: {
      phone_number_id: PHONE_NUMBER_ID,
      business_phone: "15550123456",
      webhook_secret: WEBHOOK_SECRET,
      verify_token: "verify_member_qa",
      access_token: "TEST_ACCESS_TOKEN",
      is_active: true,
    },
    orgSlug: "test-org-rbac",
    email: "admin-rbac@test.com",
  });
  const payload = buildWhatsAppTextPayload({
    phoneNumberId: PHONE_NUMBER_ID,
    from: memberPhone,
    body: `Mensaje QA miembro ${Date.now()}`,
  });
  const res = await deliverWebhook(WEBHOOK_SECRET, payload, { phoneNumberId: PHONE_NUMBER_ID });
  expect(res.status).toBe(200);
  process.env.QA_CONV_PHONE_MEMBER = memberPhone;
});

for (const vp of VIEWPORTS) {
  test(`QA admin tier ${vp.name}`, async ({ page }) => {
    await page.setViewportSize({ width: vp.width, height: vp.height });
    const consoleErrors: string[] = [];
    page.on("console", (m) => { if (m.type() === "error") consoleErrors.push(m.text().slice(0, 140)); });
    page.on("pageerror", (e) => consoleErrors.push("PAGEERROR: " + e.message.slice(0, 200)));
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": ADMIN });
    await page.goto("/dashboard/inbox");
    await expect(page.getByRole("heading", { name: "Mensajes" })).toBeVisible();
    // Open the seeded conversation to capture the composer + header.
    const phone = process.env.QA_CONV_PHONE_ADMIN;
    if (phone) {
      await page.locator(`button:has-text("${phone}")`).first().click();
    }
    await page.waitForTimeout(800);
    await page.screenshot({ path: path.join(QA_DIR, `admin-${vp.name}.png`), fullPage: false });

    // Admin controls present (Close/Reopen, writing-assist sparkles).
    const closeButtons = await page.getByRole("button", { name: /Close|Reopen/ }).count();
    expect(closeButtons).toBeGreaterThan(0);
    const sparkles = await page.locator('button[aria-label*="asistente"], button[title*="Créditos"]').count();
    report.push({
      page: "/dashboard/inbox (admin)",
      viewport: vp.name,
      closeReopenButtons: closeButtons,
      consoleErrors: consoleErrors.filter((e) => !e.includes("stytch.com") && !e.includes("ERR_FAILED")),
    });
  });
}

for (const vp of VIEWPORTS) {
  test(`QA member tier ${vp.name}`, async ({ page }) => {
    await page.setViewportSize({ width: vp.width, height: vp.height });
    const consoleErrors: string[] = [];
    page.on("console", (m) => { if (m.type() === "error") consoleErrors.push(m.text().slice(0, 140)); });
    page.on("pageerror", (e) => consoleErrors.push("PAGEERROR: " + e.message.slice(0, 200)));
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": MEMBER });
    await page.goto("/dashboard/inbox");
    await expect(page.getByRole("heading", { name: "Mensajes" })).toBeVisible();
    const phone = process.env.QA_CONV_PHONE_MEMBER;
    if (phone) {
      await page.locator(`button:has-text("${phone}")`).first().click();
    }
    await page.waitForTimeout(800);
    await page.screenshot({ path: path.join(QA_DIR, `member-${vp.name}.png`), fullPage: false });

    // Member tier: NO admin controls (close/reopen hidden, no suggestions),
    // manual composer surface available, live-region present.
    const closeButtons = await page.getByRole("button", { name: /Close|Reopen/ }).count();
    expect(closeButtons).toBe(0);
    const composer = await page.getByPlaceholder(/Escribe un mensaje|Selecciona una conversación/).count();
    expect(composer).toBeGreaterThan(0);
    const liveRegion = await page.locator('[aria-live="polite"]').count();
    expect(liveRegion).toBeGreaterThan(0);

    report.push({
      page: "/dashboard/inbox (member)",
      viewport: vp.name,
      closeReopenButtons: closeButtons,
      manualComposer: composer,
      liveRegions: liveRegion,
      consoleErrors: consoleErrors.filter((e) => !e.includes("stytch.com") && !e.includes("ERR_FAILED")),
    });
  });
}

test("write QA report", () => {
  fs.writeFileSync(path.join(QA_DIR, "qa-report.json"), JSON.stringify(report, null, 2));
});
