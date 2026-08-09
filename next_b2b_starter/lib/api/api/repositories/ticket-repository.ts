import { apiClient } from "../client/api-client";

export interface TicketDto {
  id: number;
  organization_id: number;
  contact_id?: number | null;
  conversation_id?: number | null;
  title: string;
  description?: string;
  status: "open" | "in_progress" | "waiting_customer" | "resolved" | "cancelled";
  priority: "low" | "normal" | "high";
  tags: string[];
  assignee?: string;
  sla_due_at?: string | null;
  overdue: boolean;
  created_at: string;
  updated_at: string;
}

export interface TicketEventDto {
  id: number;
  event_type: string;
  actor?: string;
  payload: Record<string, unknown>;
  created_at: string;
}

export interface TicketDetailDto {
  ticket: TicketDto;
  eventos: TicketEventDto[];
}

const BASE = "/tickets";

export const ticketRepository = {
  list: (params?: { status?: string; assignee?: string; limit?: number; offset?: number }) =>
    apiClient.get<TicketDto[]>(`${BASE}`, { params }),
  get: (id: number) => apiClient.get<TicketDetailDto>(`${BASE}/${id}`),
  create: (data: {
    contact_id?: number | null;
    conversation_id?: number | null;
    title: string;
    description?: string;
    priority?: string;
    tags?: string[];
  }) => apiClient.post<TicketDto>(`${BASE}`, data),
  transition: (id: number, status: TicketDto["status"]) =>
    apiClient.put<TicketDto>(`${BASE}/${id}/estado`, { status }),
  assign: (id: number, assignee: string) =>
    apiClient.put<TicketDto>(`${BASE}/${id}/asignacion`, { assignee }),
  setPriority: (id: number, priority: TicketDto["priority"]) =>
    apiClient.put<TicketDto>(`${BASE}/${id}/prioridad`, { priority }),
  setTags: (id: number, tags: string[]) => apiClient.put<TicketDto>(`${BASE}/${id}/etiquetas`, { tags }),
  addInternalNote: (id: number, body: string) =>
    apiClient.post<TicketEventDto>(`${BASE}/${id}/notas`, { body }),
};
