import { test, expect } from "@playwright/test";
import path from "path";
import fs from "fs";

// Visual + a11y QA for knowledge-doc-permissions.
// Artifacts → openspec/changes/knowledge-doc-permissions/qa/
const QA_DIR = path.resolve(
  __dirname,
  "../../../openspec/changes/knowledge-doc-permissions/qa"
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

test("seed an admin_only document for QA", async ({ request }) => {
  const sample = path.join(__dirname, "..", "fixtures", "sample.pdf");
  const buf = fs.readFileSync(sample);

  const upload = await request.post("http://localhost:8080/api/example_documents/upload", {
    headers: { "X-Test-Org-ID": ADMIN },
    multipart: {
      file: { name: "muestra-qa.pdf", mimeType: "application/pdf", buffer: buf },
      title: "Documento QA admin_only",
    },
  });
  expect(upload.status()).toBe(201);
  const doc = await upload.json();
  const id: number = doc.id;

  // Mark it admin_only so the member-restricted state is exercisable.
  const patch = await request.patch(`http://localhost:8080/api/example_documents/${id}`, {
    headers: { "X-Test-Org-ID": ADMIN, "Content-Type": "application/json" },
    data: { visibility: "admin_only" },
  });
  expect(patch.status()).toBe(200);

  // Member must NOT see it (404 — no title leak).
  const memberGet = await request.get(`http://localhost:8080/api/example_documents/${id}`, {
    headers: { "X-Test-Org-ID": MEMBER },
  });
  expect(memberGet.status()).toBe(404);

  // Admin must see it.
  const adminGet = await request.get(`http://localhost:8080/api/example_documents/${id}`, {
    headers: { "X-Test-Org-ID": ADMIN },
  });
  expect(adminGet.status()).toBe(200);

  test.info().attach("seeded-doc-id", { body: String(id) });
  process.env.QA_DOC_ID = String(id);
});

for (const vp of VIEWPORTS) {
  test(`QA chat mode ${vp.name}`, async ({ page }) => {
    await page.setViewportSize({ width: vp.width, height: vp.height });
    const consoleErrors: string[] = [];
    page.on("console", (m) => { if (m.type() === "error") consoleErrors.push(m.text().slice(0, 160)); });
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": ADMIN });
    await page.goto("/dashboard/knowledge?mode=chat");
    await expect(page.getByRole("button", { name: "Documentos" })).toBeVisible();
    await page.waitForTimeout(400);
    await page.screenshot({ path: path.join(QA_DIR, `chat-${vp.name}.png`), fullPage: false });

    // a11y: streaming container is a live region; composer is a labeled textarea.
    const liveRegions = await page.locator('[aria-live="polite"]').count();
    const composer = await page.getByRole("textbox").count();
    expect(liveRegions).toBeGreaterThan(0);
    expect(composer).toBeGreaterThan(0);

    report.push({
      page: "/dashboard/knowledge?mode=chat",
      viewport: vp.name,
      liveRegions,
      composerInputs: composer,
      consoleErrors: consoleErrors.filter((e) => !e.includes("stytch.com") && !e.includes("ERR_FAILED")),
    });
  });
}

for (const vp of VIEWPORTS) {
  test(`QA docs mode + detail ${vp.name}`, async ({ page }) => {
    await page.setViewportSize({ width: vp.width, height: vp.height });
    const consoleErrors: string[] = [];
    page.on("console", (m) => { if (m.type() === "error") consoleErrors.push(m.text().slice(0, 160)); });
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": ADMIN });
    await page.goto("/dashboard/knowledge?mode=docs");
    await expect(page.getByRole("button", { name: "Documentos" })).toBeVisible();
    await page.waitForTimeout(400);
    await page.screenshot({ path: path.join(QA_DIR, `docs-${vp.name}.png`), fullPage: false });

    const docId = process.env.QA_DOC_ID;
    if (docId) {
      await page.goto(`/dashboard/knowledge?mode=docs&doc=${docId}`);
      await expect(page.getByRole("button", { name: "Reprocesar" })).toBeVisible();
      await page.waitForTimeout(400);
      await page.screenshot({ path: path.join(QA_DIR, `detail-${vp.name}.png`), fullPage: false });
    }

    report.push({ page: "/dashboard/knowledge?mode=docs", viewport: vp.name, consoleErrors });
  });
}

for (const vp of VIEWPORTS) {
  test(`QA restricted state ${vp.name}`, async ({ page }) => {
    await page.setViewportSize({ width: vp.width, height: vp.height });
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": MEMBER });
    const docId = process.env.QA_DOC_ID;
    await page.goto(`/dashboard/knowledge?mode=docs&doc=${docId ?? "999999"}`);
    // Restricted state must not leak the document title.
    await expect(page.getByText("Documento restringido")).toBeVisible();
    await expect(page.getByText("Documento QA admin_only")).toHaveCount(0);
    await page.waitForTimeout(400);
    await page.screenshot({ path: path.join(QA_DIR, `restricted-${vp.name}.png`), fullPage: false });
    report.push({ page: "restricted detail", viewport: vp.name, titleLeaked: false });
  });
}

test("write QA report", () => {
  fs.writeFileSync(path.join(QA_DIR, "qa-report.json"), JSON.stringify(report, null, 2));
});
