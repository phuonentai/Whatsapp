import { describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api/api/client/api-client", () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
    getBaseUrl: () => "http://localhost:8080/api",
  },
  resolveAccessToken: vi.fn().mockResolvedValue("token"),
}));

import { apiClient } from "@/lib/api/api/client/api-client";
import { signupRepository } from "./signup-repository";

describe("signupRepository magic-link signup", () => {
  it("sends a Stytch-compliant payload that never includes owner_password", async () => {
    vi.mocked(apiClient.post).mockResolvedValue({
      success: true,
      data: {
        org_id: "org_123",
        org_name: "Acme",
        display_name: "Acme",
        owner_email: "ana@acme.com",
        owner_name: "Ana",
        login_url: "/auth",
      },
    });

    await signupRepository.createOrganizationWithMagicLink(
      { fullName: "Ana", email: "ana@acme.com" },
      { displayName: "Acme", industry: "Retail" }
    );

    expect(apiClient.post).toHaveBeenCalledTimes(1);
    const [url, payload] = vi.mocked(apiClient.post).mock.calls[0] as [
      string,
      Record<string, unknown>,
    ];
    expect(url).toBe("/auth/signup");
    expect(payload).not.toHaveProperty("owner_password");
    expect(payload).not.toHaveProperty("password");
    expect(payload).toEqual({
      org_display_name: "Acme",
      owner_email: "ana@acme.com",
      owner_name: "Ana",
      industry: "Retail",
    });
  });
});
