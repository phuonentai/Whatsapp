import { apiClient } from "../client/api-client";
import { WhatsAppConfigDto } from "../dto/whatsapp-config.dto";
import { WhatsAppConfig, WhatsAppConfigInput } from "@/lib/models/whatsapp-config.model";

class WhatsAppConfigRepository {
  async getConfig(): Promise<WhatsAppConfig> {
    const response = await apiClient.get<WhatsAppConfigDto>("/whatsapp/config");
    return this.toModel(response);
  }

  async upsertConfig(input: WhatsAppConfigInput): Promise<WhatsAppConfig> {
    const response = await apiClient.put<WhatsAppConfigDto>("/whatsapp/config", input);
    return this.toModel(response);
  }

  async toggleConfig(): Promise<WhatsAppConfig> {
    const response = await apiClient.patch<WhatsAppConfigDto>("/whatsapp/config/toggle");
    return this.toModel(response);
  }

  private toModel(dto: WhatsAppConfigDto): WhatsAppConfig {
    return {
      id: dto.id,
      organizationId: dto.organization_id,
      phoneNumberId: dto.phone_number_id,
      businessPhone: dto.business_phone,
      webhookSecret: dto.webhook_secret,
      verifyToken: dto.verify_token,
      appId: dto.app_id ?? undefined,
      wabaId: dto.waba_id ?? undefined,
      accessToken: dto.access_token ?? undefined,
      apiVersion: dto.api_version ?? undefined,
      graphApiUrl: dto.graph_api_url ?? undefined,
      isActive: dto.is_active,
      metadata: dto.metadata,
      createdAt: new Date(dto.created_at),
      updatedAt: new Date(dto.updated_at),
    };
  }
}

export const whatsappConfigRepository = new WhatsAppConfigRepository();
