import { apiClient } from "../client/api-client";
import {
  WhatsAppTemplate,
  WhatsAppTemplateInput,
} from "@/lib/models/whatsapp-template.model";

class WhatsAppTemplateRepository {
  async list(): Promise<WhatsAppTemplate[]> {
    return apiClient.get<WhatsAppTemplate[]>("/v1/whatsapp/templates");
  }

  async create(input: WhatsAppTemplateInput): Promise<WhatsAppTemplate> {
    return apiClient.post<WhatsAppTemplate>("/v1/whatsapp/templates", input);
  }

  async update(
    id: number,
    input: WhatsAppTemplateInput
  ): Promise<WhatsAppTemplate> {
    return apiClient.patch<WhatsAppTemplate>(
      `/v1/whatsapp/templates/${id}`,
      input
    );
  }

  async remove(id: number): Promise<void> {
    await apiClient.delete(`/v1/whatsapp/templates/${id}`);
  }

  async submit(id: number): Promise<WhatsAppTemplate> {
    return apiClient.post<WhatsAppTemplate>(
      `/v1/whatsapp/templates/${id}/submit`
    );
  }

  async sync(id: number): Promise<WhatsAppTemplate> {
    return apiClient.post<WhatsAppTemplate>(`/v1/whatsapp/templates/${id}/sync`);
  }
}

export const whatsappTemplateRepository = new WhatsAppTemplateRepository();
