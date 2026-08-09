import { apiClient } from "../client/api-client";
import {
  WhatsAppSignupMetaConfigDto,
  WhatsAppSignupResultDto,
  WhatsAppSignupStatusDto,
} from "../dto/whatsapp-signup.dto";
import {
  WhatsAppSignupMetaConfig,
  WhatsAppSignupResult,
  WhatsAppSignupStatus,
} from "@/lib/models/whatsapp-signup.model";

class WhatsAppSignupRepository {
  async getMetaConfig(): Promise<WhatsAppSignupMetaConfig> {
    const response = await apiClient.get<WhatsAppSignupMetaConfigDto>(
      "/v1/whatsapp/signup/meta-config"
    );
    return {
      app_id: response.app_id,
      config_id: response.config_id,
      redirect_uri: response.redirect_uri,
    };
  }

  async exchange(code: string): Promise<WhatsAppSignupResult> {
    const response = await apiClient.post<WhatsAppSignupResultDto>(
      "/v1/whatsapp/signup/exchange",
      { code }
    );
    return {
      status: response.status as WhatsAppSignupResult["status"],
      error_code: response.error_code,
      config: response.config
        ? {
            phone_number_id: response.config.phone_number_id,
            business_phone: response.config.business_phone,
            app_id: response.config.app_id,
            waba_id: response.config.waba_id,
            is_active: response.config.is_active,
          }
        : undefined,
    };
  }

  async getStatus(): Promise<WhatsAppSignupStatus> {
    const response = await apiClient.get<WhatsAppSignupStatusDto>(
      "/v1/whatsapp/signup/status"
    );
    return {
      id: response.id,
      organization_id: response.organization_id,
      status: response.status as WhatsAppSignupStatus["status"],
      step: response.step,
      error_code: response.error_code,
      retry_count: response.retry_count,
      metadata: response.metadata,
      created_at: response.created_at,
      updated_at: response.updated_at,
    };
  }
}

export const whatsappSignupRepository = new WhatsAppSignupRepository();
