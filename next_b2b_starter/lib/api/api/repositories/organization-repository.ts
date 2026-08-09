// lib/api/api/repositories/organization-repository.ts

import { apiClient } from "../client/api-client";

export interface UpdateOrganizationPayload {
  name: string;
  status: string;
}

class OrganizationRepository {
  /**
   * Update the current organization (workspace) metadata.
   * Requires org:manage permission. Mirrors PUT /api/organizations.
   */
  async updateOrganization(payload: UpdateOrganizationPayload): Promise<void> {
    await apiClient.put("/organizations", payload);
  }
}

export const organizationRepository = new OrganizationRepository();
