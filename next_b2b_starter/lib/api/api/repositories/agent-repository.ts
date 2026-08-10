// lib/api/api/repositories/agent-repository.ts

import { apiClient } from "../client/api-client";
import {
  AgentSettingsDto,
  AgentSuggestionDto,
  ComplianceExportDto,
} from "../dto/agent.dto";
import type {
  AgentSettings,
  AgentSuggestion,
  ComplianceExport,
} from "@/lib/models/agent.model";

interface Wrapped<T> {
  success: boolean;
  data: T;
}

class AgentRepository {
  async getSettings(): Promise<AgentSettings> {
    const response = await apiClient.get<Wrapped<AgentSettingsDto>>("/agent/settings");
    return this.toSettingsModel(response.data);
  }

  async updateSettings(input: Partial<AgentSettings>): Promise<AgentSettings> {
    const response = await apiClient.put<Wrapped<AgentSettingsDto>>(
      "/agent/settings",
      input
    );
    return this.toSettingsModel(response.data);
  }

  async listSuggestions(): Promise<AgentSuggestion[]> {
    const response = await apiClient.get<Wrapped<{ suggestions: AgentSuggestionDto[] }>>(
      "/agent/suggestions?status=pending"
    );
    const suggestions = response.data?.suggestions ?? (response as unknown as { suggestions?: AgentSuggestionDto[] }).suggestions ?? [];
    return suggestions.map(this.toSuggestionModel);
  }

  async approveSuggestion(
    suggestionId: number,
    editedBody?: string
  ): Promise<AgentSuggestion> {
    const response = await apiClient.post<Wrapped<AgentSuggestionDto>>(
      `/agent/suggestions/${suggestionId}/approve`,
      editedBody ? { edited_body: editedBody } : {}
    );
    return this.toSuggestionModel(response.data);
  }

  async rejectSuggestion(suggestionId: number): Promise<AgentSuggestion> {
    const response = await apiClient.post<Wrapped<AgentSuggestionDto>>(
      `/agent/suggestions/${suggestionId}/reject`,
      {}
    );
    return this.toSuggestionModel(response.data);
  }

  async exportContact(contactId: number): Promise<ComplianceExport> {
    const response = await apiClient.get<Wrapped<ComplianceExportDto>>(
      `/agent/compliance/export/${contactId}`
    );
    return this.toExportModel(response.data);
  }

  async forgetContact(contactId: number): Promise<void> {
    await apiClient.post<Wrapped<{ status: string }>>(
      `/agent/compliance/forget/${contactId}`,
      {}
    );
  }

  private toSettingsModel(dto: AgentSettingsDto): AgentSettings {
    return {
      id: dto.id,
      organization_id: dto.organization_id,
      mode: dto.mode === "autopilot" ? "autopilot" : "copilot",
      tone: dto.tone === "casual" ? "casual" : "formal",
      brand_voice: dto.brand_voice ?? "",
      autopilot_start: dto.autopilot_start ?? "",
      autopilot_end: dto.autopilot_end ?? "",
      timezone: dto.timezone ?? "America/Bogota",
      kill_switch: dto.kill_switch,
      max_daily_messages: dto.max_daily_messages,
      consent_required: dto.consent_required,
      consent_template: dto.consent_template ?? "",
      guardrails: dto.guardrails ?? {},
    };
  }

  private toSuggestionModel(dto: AgentSuggestionDto): AgentSuggestion {
    return {
      id: dto.id,
      organization_id: dto.organization_id,
      conversation_id: dto.conversation_id,
      contact_id: dto.contact_id,
      flow_id: dto.flow_id,
      type: dto.type as AgentSuggestion["type"],
      body: dto.body,
      status: dto.status as AgentSuggestion["status"],
      source: dto.source as AgentSuggestion["source"],
      approved_by_member_id: dto.approved_by_member_id,
      whatsapp_message_id: dto.whatsapp_message_id,
      created_at: dto.created_at,
      updated_at: dto.updated_at,
    };
  }

  private toExportModel(dto: ComplianceExportDto): ComplianceExport {
    return dto as ComplianceExport;
  }
}

export const agentRepository = new AgentRepository();
