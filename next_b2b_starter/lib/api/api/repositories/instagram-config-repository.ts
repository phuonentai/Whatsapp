import { apiClient } from "../client/api-client";
import { InstagramConfigDto, InstagramWebhookHealthDto } from "../dto/instagram-config.dto";
import { InstagramConfig, InstagramConfigInput } from "@/lib/models/instagram-config.model";

class InstagramConfigRepository {
  async getConfig(): Promise<InstagramConfig> {
    const response = await apiClient.get<InstagramConfigDto>("/v1/instagram/config");
    return this.toModel(response);
  }

  async upsertConfig(input: InstagramConfigInput): Promise<InstagramConfig> {
    const response = await apiClient.put<InstagramConfigDto>("/v1/instagram/config", input);
    return this.toModel(response);
  }

  async toggleConfig(): Promise<InstagramConfig> {
    const response = await apiClient.patch<InstagramConfigDto>("/v1/instagram/config/toggle");
    return this.toModel(response);
  }

  async refreshToken(): Promise<InstagramConfig> {
    const response = await apiClient.post<InstagramConfigDto>("/v1/instagram/config/refresh");
    return this.toModel(response);
  }

  async getHealth(): Promise<InstagramWebhookHealthDto> {
    return apiClient.get<InstagramWebhookHealthDto>("/v1/instagram/config/health");
  }

  private toModel(dto: InstagramConfigDto): InstagramConfig {
    return {
      id: dto.id,
      organizationId: dto.organization_id,
      igUserId: dto.ig_user_id,
      igUsername: dto.ig_username ?? undefined,
      fbPageId: dto.fb_page_id ?? undefined,
      accessToken: dto.access_token ?? undefined,
      tokenExpiresAt: dto.token_expires_at ?? undefined,
      tokenExpiryWarning: dto.token_expiry_warning ?? false,
      webhookSecret: dto.webhook_secret,
      verifyToken: dto.verify_token,
      apiVersion: dto.api_version ?? undefined,
      graphApiUrl: dto.graph_api_url ?? undefined,
      isActive: dto.is_active,
      metadata: dto.metadata,
      createdAt: dto.created_at,
      updatedAt: dto.updated_at,
    };
  }
}

export const instagramConfigRepository = new InstagramConfigRepository();
