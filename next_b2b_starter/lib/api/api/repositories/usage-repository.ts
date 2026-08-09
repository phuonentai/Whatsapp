// lib/api/api/repositories/usage-repository.ts

import { apiClient } from "../client/api-client";
import { AiUsageDto } from "../dto/ai-usage.dto";

class UsageRepository {
  /**
   * Get the current-period AI usage for the authenticated organization.
   */
  async getAiUsage(): Promise<AiUsageDto> {
    return apiClient.get<AiUsageDto>("/crm/usage/ai");
  }
}

export const usageRepository = new UsageRepository();
