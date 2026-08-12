import { beforeEach, describe, expect, it, vi } from "vitest";

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
import { rbacRepository } from "./rbac-repository";

describe("rbacRepository", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetches roles with the session attached (no skipAuth bypass)", async () => {
    vi.mocked(apiClient.get).mockResolvedValue({
      roles: [
        {
          id: "admin",
          name: "Admin",
          description: "Full access",
          typical_users: "Admins",
          permissions: [],
        },
      ],
    });

    const roles = await rbacRepository.getRoles(true);

    expect(roles).toHaveLength(1);
    expect(roles[0].id).toBe("admin");
    expect(roles[0].name).toBe("Admin");
    expect(roles[0].typicalUsers).toBe("Admins");
    expect(apiClient.get).toHaveBeenCalledTimes(1);

    const [url, options] = vi.mocked(apiClient.get).mock.calls[0] as [
      string,
      Record<string, unknown> | undefined,
    ];
    expect(url).toBe("/rbac/roles");
    // RBAC endpoints require an authenticated session; the request must NOT
    // carry skipAuth (the client attaches the access token when it is unset).
    expect(options).toBeUndefined();
  });

  it("caches roles without refetching within a session", async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ roles: [] });

    await rbacRepository.getRoles(true); // prime the in-memory cache
    await rbacRepository.getRoles(); // cache hit

    expect(apiClient.get).toHaveBeenCalledTimes(1);
  });
});
