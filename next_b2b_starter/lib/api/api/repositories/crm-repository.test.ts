import { describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api/api/client/api-client", () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
    getBaseUrl: () => "http://localhost:8080/api",
  },
  resolveAccessToken: vi.fn().mockResolvedValue("token"),
}));

import { apiClient } from "@/lib/api/api/client/api-client";
import { crmRepository } from "@/lib/api/api/repositories/crm-repository";
import type { ContactDto, CompanyDto } from "@/lib/api/api/dto/crm.dto";

const contact: ContactDto = {
  id: 1,
  organization_id: 42,
  display_name: "Ana",
  phone_number: "573001234567",
  source: "whatsapp",
  lead_status: "nuevo",
  is_blocked: false,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

describe("crmRepository paginated list unwrapping", () => {
  it("exposes total alongside data for listContacts", async () => {
    vi.mocked(apiClient.get).mockResolvedValue({
      success: true,
      data: [contact],
      total: 137,
    });

    const result = await crmRepository.listContacts({ limit: 25, offset: 0 });

    expect(result.items).toEqual([contact]);
    expect(result.total).toBe(137);
  });

  it("falls back to total 0 when the envelope has no total", async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ success: true, data: [contact] });

    const result = await crmRepository.listContacts();

    expect(result.items).toEqual([contact]);
    expect(result.total).toBe(0);
  });

  it("exposes total for listCompanies", async () => {
    const company: CompanyDto = {
      id: 1,
      organization_id: 42,
      name: "Tienda",
      nit: "900000001",
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    };
    vi.mocked(apiClient.get).mockResolvedValue({
      success: true,
      data: [company],
      total: 4,
    });

    const result = await crmRepository.listCompanies({ limit: 25, offset: 0 });

    expect(result.items).toEqual([company]);
    expect(result.total).toBe(4);
  });
});
