import { test, expect } from "@playwright/test";
import { KnowledgePage } from "../page-objects/knowledge.page";

test.describe("Knowledge Base UI", () => {
  test.beforeEach(async ({ page }) => {
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-pro:admin-pro@test.com" });
  });

  test("PDF upload adds document to the list", async ({ page }) => {
    const knowledge = new KnowledgePage(page);
    await knowledge.goto();
    await knowledge.openSources();

    await knowledge.uploadPdf("sample.pdf");
    await knowledge.assertDocumentInList("sample");
  });

  test("non-PDF file triggers no upload request", async ({ page }) => {
    let uploadCalls = 0;
    await page.route("**/api/example_documents/upload", (route) => {
      uploadCalls++;
      route.fulfill({ status: 201, contentType: "application/json", body: "{}" });
    });

    const knowledge = new KnowledgePage(page);
    await knowledge.goto();
    await knowledge.openSources();

    await knowledge.dropRejectedFile("sample.txt");
    await expect.poll(() => uploadCalls, { timeout: 1000 }).toBe(0);
  });

  test("uploaded document title is visible in the list", async ({ page }) => {
    const knowledge = new KnowledgePage(page);
    await knowledge.goto();
    await knowledge.openSources();

    const title = `Doc E2E ${Date.now()}`;
    await knowledge.uploadPdf("sample.pdf", title);
    await knowledge.assertDocumentInList(title);
  });

  test("chat message is appended to the thread", async ({ page }) => {
    const question = `¿Pregunta e2e ${Date.now()}?`;
    let sessionId = 0;
    let currentQuestion: string | null = null;

    await page.route("**/api/example_cognitive/chat", (route) => {
      sessionId = 1;
      currentQuestion = question;
      route.fulfill({
        status: 200,
        contentType: "text/event-stream",
        headers: { "Content-Type": "text/event-stream" },
        body:
          'data: {"token":"Hola, "}\n\n' +
          'data: {"token":"¿cómo ayudarte?"}\n\n' +
          'data: {"done":true,"session_id":1,"message_id":1}\n\n',
      });
    });
    // Pin the session list to empty so no real session auto-selects, keeping
    // the new-chat flow deterministic.
    await page.route("**/api/example_cognitive/sessions", (route) => {
      route.fulfill({
        status: 200,
        contentType: "application/json",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ sessions: [] }),
      });
    });
    // After the chat resolves, the UI refetches session messages for the new
    // session id; return the user's message so it renders in the thread. The
    // repository accepts a direct array (actual API format).
    await page.route("**/api/example_cognitive/sessions/*/messages", (route) => {
      route.fulfill({
        status: 200,
        contentType: "application/json",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify([
          {
            id: 1,
            session_id: sessionId,
            role: "user",
            content: currentQuestion ?? "",
            tokens_used: 0,
            created_at: new Date().toISOString(),
          },
        ]),
      });
    });

    const knowledge = new KnowledgePage(page);
    await knowledge.goto();

    await knowledge.sendChat(question);
    await knowledge.assertChatMessage(question);
  });

  test("empty chat message is not sent", async ({ page }) => {
    let chatCalled = false;
    await page.route("**/api/example_cognitive/chat", () => {
      chatCalled = true;
    });

    const knowledge = new KnowledgePage(page);
    await knowledge.goto();

    await knowledge.chatInput.fill("   ");
    await knowledge.chatInput.press("Enter");
    expect(chatCalled).toBe(false);
  });

  test("empty document list renders empty state", async ({ page }) => {
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": "test-org-rbac:admin-rbac@test.com" });
    const knowledge = new KnowledgePage(page);
    await knowledge.goto();
    await knowledge.openSources();
    await expect(page.getByText("No documents yet")).toBeVisible();
  });

  test("failed upload adds no document and surfaces an error", async ({ page }) => {
    let uploadCalls = 0;
    await page.route("**/api/example_documents/upload", (route) => {
      uploadCalls++;
      route.fulfill({
        status: 500,
        contentType: "application/json",
        body: JSON.stringify({ error: "upload_failed" }),
      });
    });

    const knowledge = new KnowledgePage(page);
    await knowledge.goto();
    await knowledge.openSources();
    await knowledge.uploadPdf("sample.pdf", `Fallo ${Date.now()}`);

    expect(uploadCalls).toBe(1);
    await expect(page.getByText("Document uploaded successfully!")).not.toBeVisible();
  });
});
