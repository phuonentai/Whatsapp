import { test, expect } from "@playwright/test";
import { SettingsPage } from "../page-objects/settings.page";
import { AdminPanelPage } from "../page-objects/admin-panel.page";
import { apiRequest } from "../helpers/api";

/**
 * Siigo onboarding e2e suite.
 *
 * Runs serially against the dedicated `test-org-siigo` org (seeded with
 * admin-siigo@test.com / member-siigo@test.com) so connection state built by
 * earlier scenarios is reused by later ones, and general-purpose orgs stay
 * clean. The backend points at the mock Siigo provider (cmd/mock-siigo) via
 * SIIGO_BASE_URL/SIIGO_TOKEN_URL exported by scripts/run_e2e.sh — no real
 * Siigo traffic.
 */

const SIIGO_ADMIN = "test-org-siigo:admin-siigo@test.com";
const SIIGO_MEMBER = "test-org-siigo:member-siigo@test.com";
const PRO_ORG = "test-org-pro:admin-pro@test.com";

const MOCK_CREDS = {
  clientId: "e2e-client",
  clientSecret: "e2e-secret",
  nit: "900123456",
};

test.describe.serial("Siigo onboarding", () => {
  test.beforeEach(async ({ page }) => {
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": SIIGO_ADMIN });
  });

  test("assisted setup: admin requests, admin provisions", async ({ page }) => {
    const settings = new SettingsPage(page);
    await settings.openSiigoSection();

    // The suite presumes a fresh DB (`make test-e2e` resets it), but ad-hoc
    // re-runs against a live stack inherit connection state. Drive the flow
    // from the backend status instead of the UI banner so any leftover state
    // is tolerated.
    const status = (await getSiigoStatus(SIIGO_ADMIN)) as string;

    if (status === "none") {
      await settings.assertSiigoBanner("Conecta Siigo para facturar");

      // Non-admin member is gated out: no Siigo section entry (overview only).
      await page.setExtraHTTPHeaders({ "X-Test-Org-ID": SIIGO_MEMBER });
      await page.reload();
      await expect(page.getByText("Integración Siigo")).toHaveCount(0);

      // Admin requests assisted setup.
      await page.setExtraHTTPHeaders({ "X-Test-Org-ID": SIIGO_ADMIN });
      await page.reload();
      await settings.requestAssistedSetup();
      await settings.assertSiigoBanner("Tu equipo está configurando tu facturación");
    } else if (status === "awaiting_setup") {
      await settings.assertSiigoBanner("Tu equipo está configurando tu facturación");
    }

    // Provision when not yet connected.
    if (status === "none" || status === "awaiting_setup") {
      const admin = new AdminPanelPage(page);
      await admin.openSiigoOnboarding();
      await admin.assertSiigoRow("5", "Esperando configuración");
      await admin.provisionCredentials(MOCK_CREDS);
    }

    // Client section shows onboarding progress (any step past connect) —
    // tolerant to leftover states from previous ad-hoc runs. A retry that
    // runs after the wizard scenario inherits a `live` org, which renders
    // "Facturación activa" instead of a numbered step.
    await page.goto("/dashboard/settings?view=siigo");
    await expect(page.getByText(/Paso [2-5]|Facturación activa/).first()).toBeVisible();
  });

  test("wizard happy path: numeración → import → sandbox → activar", async ({ page }) => {
    const settings = new SettingsPage(page);
    await settings.openSiigoSection();

    // State-driven: a leftover state from a previous ad-hoc run skips the
    // steps already completed (the suite presumes a fresh DB via
    // `make test-e2e`; this tolerance only covers manual re-runs).
    let status = (await getSiigoStatus(SIIGO_ADMIN)) as string;

    // Step 2 — numeration (auto mode, mock has no numeration resource).
    if (status === "connected" || status === "none" || status === "awaiting_setup") {
      await settings.confirmNumeration();
      await settings.assertSiigoBanner("Paso 3 — Importa tus clientes");
      status = (await getSiigoStatus(SIIGO_ADMIN)) as string;
    }

    // Step 3 — import preview + confirm (4 mock customers, all nuevos).
    if (status === "numeracion_ok") {
      await settings.previewImport();
      await settings.assertSiigoBanner("4 clientes encontrados");
      await settings.confirmImport();
      await settings.assertSiigoBanner(/Importación completada: \d+ nuevos · \d+ existentes/);
      status = (await getSiigoStatus(SIIGO_ADMIN)) as string;
    }

    // Step 4 — sandbox test invoice (mock returns valid immediately).
    if (status === "sandbox_ok" || status === "numeracion_ok") {
      await settings.createTestInvoice();
      await settings.assertSiigoBanner(/Factura .* — estado: valid/);
      status = (await getSiigoStatus(SIIGO_ADMIN)) as string;
    }

    // Step 5 — activate.
    if (status === "sandbox_ok") {
      await settings.activateInvoicing();
    }
    await settings.assertSiigoBanner("Facturación activa");
  });

  test("kill-switch: pause and resume from settings", async ({ page }) => {
    const settings = new SettingsPage(page);
    await settings.openSiigoSection();

    await settings.pauseInvoicing();
    await settings.assertSiigoBanner("Facturación pausada");

    await settings.resumeInvoicing();
    await settings.assertSiigoBanner("Facturación activa");
  });

  test("admin view shows the onboarded organization", async ({ page }) => {
    const admin = new AdminPanelPage(page);
    await admin.openSiigoOnboarding();
    const row = await admin.assertSiigoRow("5", "Activo");
    // Mock has no numeration resource → auto mode snapshot shown.
    await expect(row.getByText("auto")).toBeVisible();
    await expect(row.getByText(/confirm · \d+ nuevos/)).toBeVisible();
  });

  test("gating: deal to facturado invoices only when live", async ({ page }) => {
    const refs = await setupFacturadoPipeline(SIIGO_ADMIN, "E2E Siigo Live");
    const liveDealId = await createDealAndMoveToFacturado(SIIGO_ADMIN, refs, "Negocio Live", 250000);

    // Live org: invoice created and resolved valid by the mock.
    await expect
      .poll(() => dealActivities(SIIGO_ADMIN, liveDealId), { timeout: 15_000 })
      .toContain("Factura electrónica creada");

    // Fresh org (pro, no connection): no invoice, inactive activity instead.
    const freshRefs = await setupFacturadoPipeline(PRO_ORG, "E2E Siigo Fresh");
    const freshDealId = await createDealAndMoveToFacturado(PRO_ORG, freshRefs, "Negocio Fresco", 100000);
    await expect
      .poll(() => dealActivities(PRO_ORG, freshDealId), { timeout: 15_000 })
      .toContain("Facturación no activa");
  });

  test("isolation: fresh org still shows the connect invitation", async ({ page }) => {
    await page.setExtraHTTPHeaders({ "X-Test-Org-ID": PRO_ORG });
    const settings = new SettingsPage(page);
    await settings.openSiigoSection();
    await settings.assertSiigoBanner("Conecta Siigo para facturar");
  });
});

// ---- API helpers -----------------------------------------------------------

interface PipelineRefs {
  pipelineId: number;
  cotizacionId: number;
  facturadoId: number;
}

async function setupFacturadoPipeline(auth: string, name: string): Promise<PipelineRefs> {
  const [slug, email] = auth.split(":");
  const pipeline = await apiRequest<any>("/crm/pipelines", {
    method: "POST",
    orgSlug: slug,
    email,
    body: { nombre: name, orden: 0 },
  });
  const data = (pipeline as any).data ?? pipeline;
  const pipelineId = data.id;

  const stage = await apiRequest<any>(`/crm/pipelines/${pipelineId}/etapas`, {
    method: "POST",
    orgSlug: slug,
    email,
    body: { nombre: "cotizacion", orden: 0, color: "#3b82f6" },
  });
  const stageData = (stage as any).data ?? stage;
  const cotizacionId = stageData.id;

  const facturado = await apiRequest<any>(`/crm/pipelines/${pipelineId}/etapas`, {
    method: "POST",
    orgSlug: slug,
    email,
    body: { nombre: "facturado", orden: 1, color: "#10b981" },
  });
  const facturadoData = (facturado as any).data ?? facturado;

  return { pipelineId, cotizacionId, facturadoId: facturadoData.id };
}

async function createDealAndMoveToFacturado(
  auth: string,
  refs: PipelineRefs,
  nombre: string,
  monto: number,
): Promise<number> {
  const [slug, email] = auth.split(":");
  const deal = await apiRequest<any>("/crm/negocios", {
    method: "POST",
    orgSlug: slug,
    email,
    body: {
      nombre,
      pipeline_id: refs.pipelineId,
      stage_id: refs.cotizacionId,
      monto,
      moneda: "COP",
    },
  });
  const dealData = (deal as any).data ?? deal;
  const dealId = dealData.id;

  await apiRequest<any>(`/crm/negocios/${dealId}/etapa`, {
    method: "PUT",
    orgSlug: slug,
    email,
    body: {
      stage_id: refs.facturadoId,
      old_stage_name: "cotizacion",
      new_stage_name: "facturado",
    },
  });
  return dealId;
}

async function getSiigoStatus(auth: string): Promise<string> {
  const [slug, email] = auth.split(":");
  const res = await apiRequest<any>("/v1/org/siigo/status", { orgSlug: slug, email });
  const data = (res as any).data ?? res;
  return data?.status ?? "none";
}

async function dealActivities(auth: string, dealId: number): Promise<string> {
  const [slug, email] = auth.split(":");
  const res = await apiRequest<any>(`/crm/actividades/negocio/${dealId}`, {
    orgSlug: slug,
    email,
  });
  const activities = (res as any).data ?? res;
  if (!Array.isArray(activities)) return "";
  return activities.map((a: any) => a.asunto ?? a.subject ?? "").join("|");
}
