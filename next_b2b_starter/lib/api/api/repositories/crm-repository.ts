import { apiClient, resolveAccessToken } from "../client/api-client";
import type { ContactDto, CompanyDto, DealDto, PipelineDto, ActivityDto, TagDto, EntitlementDto } from "../dto/crm.dto";

const BASE = "/crm";

export interface ImportSummaryDto {
  importados: number;
  omitidos: number;
  errores: { fila: number; razon: string }[];
}

// CSV endpoints stream a file, so they cannot use the JSON unwrapping ApiClient.
// Fetch + blob carries the Stytch session token in the request headers (a bare
// window.location navigation cannot attach it).
async function downloadCSV(endpoint: string, filename: string): Promise<void> {
  const token = await resolveAccessToken();
  const headers: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {};
  const response = await fetch(`${apiClient.getBaseUrl()}${endpoint}`, {
    headers,
    credentials: "include",
  });
  if (!response.ok) {
    throw new Error(`API Error ${response.status}: no se pudo descargar el archivo`);
  }
  const blob = await response.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

// The backend wraps all CRM responses in a { data, success } envelope.
// Unwrap it so callers receive the payload directly (matches EntitlementDto
// and the component types).
type Envelope<T> = { data?: T; success?: boolean };

async function unwrap<T>(request: Promise<Envelope<T> | T>): Promise<T> {
  const response = await request;
  if (response !== null && typeof response === "object" && "data" in response) {
    return (response as Envelope<T>).data as T;
  }
  return response as T;
}

export const crmRepository = {
  getEntitlement: () => unwrap(apiClient.get<Envelope<EntitlementDto>>(`${BASE}/entitlement`)),

  listContacts: (params?: { source?: string; lead_status?: string; limit?: number; offset?: number }) =>
    unwrap(apiClient.get<Envelope<ContactDto[]>>(`${BASE}/contactos`, { params })),
  searchContacts: (q: string, params?: { limit?: number; offset?: number }) =>
    unwrap(apiClient.get<Envelope<ContactDto[]>>(`${BASE}/contactos/search`, { params: { q, ...params } })),
  getContact: (id: number) => unwrap(apiClient.get<Envelope<ContactDto>>(`${BASE}/contactos/${id}`)),
  createContact: (data: Partial<ContactDto>) => unwrap(apiClient.post<Envelope<ContactDto>>(`${BASE}/contactos`, data)),
  updateContact: (id: number, data: Partial<ContactDto>) => unwrap(apiClient.put<Envelope<ContactDto>>(`${BASE}/contactos/${id}`, data)),
  deleteContact: (id: number) => unwrap(apiClient.delete<Envelope<null>>(`${BASE}/contactos/${id}`)),

  listCompanies: (params?: { limit?: number; offset?: number }) =>
    unwrap(apiClient.get<Envelope<CompanyDto[]>>(`${BASE}/empresas`, { params })),
  searchCompanies: (q: string) => unwrap(apiClient.get<Envelope<CompanyDto[]>>(`${BASE}/empresas/search`, { params: { q } })),
  getCompany: (id: number) => unwrap(apiClient.get<Envelope<CompanyDto>>(`${BASE}/empresas/${id}`)),
  createCompany: (data: Partial<CompanyDto>) => unwrap(apiClient.post<Envelope<CompanyDto>>(`${BASE}/empresas`, data)),
  updateCompany: (id: number, data: Partial<CompanyDto>) => unwrap(apiClient.put<Envelope<CompanyDto>>(`${BASE}/empresas/${id}`, data)),
  deleteCompany: (id: number) => unwrap(apiClient.delete<Envelope<null>>(`${BASE}/empresas/${id}`)),

  listDeals: (params?: { pipeline_id?: number; stage_id?: number; estado?: string; contact_id?: number; limit?: number; offset?: number }) =>
    unwrap(apiClient.get<Envelope<DealDto[]>>(`${BASE}/negocios`, { params })),
  getDeal: (id: number) => unwrap(apiClient.get<Envelope<DealDto>>(`${BASE}/negocios/${id}`)),
  createDeal: (data: Partial<DealDto>) => unwrap(apiClient.post<Envelope<DealDto>>(`${BASE}/negocios`, data)),
  updateDeal: (id: number, data: Partial<DealDto>) => unwrap(apiClient.put<Envelope<DealDto>>(`${BASE}/negocios/${id}`, data)),
  moveDealStage: (id: number, data: { stage_id: number; old_stage_name: string; new_stage_name: string }) =>
    unwrap(apiClient.put<Envelope<DealDto>>(`${BASE}/negocios/${id}/etapa`, data)),
  deleteDeal: (id: number) => unwrap(apiClient.delete<Envelope<null>>(`${BASE}/negocios/${id}`)),

  listPipelines: () => unwrap(apiClient.get<Envelope<PipelineDto[]>>(`${BASE}/pipelines`)),
  createPipeline: (data: Partial<PipelineDto>) => unwrap(apiClient.post<Envelope<PipelineDto>>(`${BASE}/pipelines`, data)),
  updatePipeline: (id: number, data: Partial<PipelineDto>) => unwrap(apiClient.put<Envelope<PipelineDto>>(`${BASE}/pipelines/${id}`, data)),
  createStage: (pipelineId: number, data: { nombre: string; orden: number; color?: string; probabilidad?: number }) =>
    unwrap(apiClient.post<Envelope<{ id: number }>>(`${BASE}/pipelines/${pipelineId}/etapas`, data)),
  updateStage: (pipelineId: number, stageId: number, data: Partial<{ nombre: string; orden: number; color: string; probabilidad: number }>) =>
    unwrap(apiClient.put<Envelope<{ id: number }>>(`${BASE}/pipelines/${pipelineId}/etapas/${stageId}`, data)),

  listActivities: (params?: { tipo?: string; limit?: number; offset?: number }) =>
    unwrap(apiClient.get<Envelope<ActivityDto[]>>(`${BASE}/actividades`, { params })),
  createActivity: (data: Partial<ActivityDto>) => unwrap(apiClient.post<Envelope<ActivityDto>>(`${BASE}/actividades`, data)),
  listActivitiesByContact: (contactId: number) => unwrap(apiClient.get<Envelope<ActivityDto[]>>(`${BASE}/actividades/contacto/${contactId}`)),
  listActivitiesByDeal: (dealId: number) => unwrap(apiClient.get<Envelope<ActivityDto[]>>(`${BASE}/actividades/negocio/${dealId}`)),
  listActivitiesByCompany: (companyId: number) => unwrap(apiClient.get<Envelope<ActivityDto[]>>(`${BASE}/actividades/empresa/${companyId}`)),

  listTags: () => unwrap(apiClient.get<Envelope<TagDto[]>>(`${BASE}/etiquetas`)),
  createTag: (data: { nombre: string; color?: string }) => unwrap(apiClient.post<Envelope<TagDto>>(`${BASE}/etiquetas`, data)),
  updateTag: (id: number, data: { nombre: string; color?: string }) =>
    unwrap(apiClient.put<Envelope<TagDto>>(`${BASE}/etiquetas/${id}`, data)),
  deleteTag: (id: number) => unwrap(apiClient.delete<Envelope<null>>(`${BASE}/etiquetas/${id}`)),
  listEntityTags: (entityType: string, entityId: number) =>
    unwrap(apiClient.get<Envelope<TagDto[]>>(`${BASE}/etiquetas/entity/${entityType}/${entityId}`)),
  tagEntity: (entityType: string, entityId: number, tagId: number) =>
    unwrap(apiClient.post<Envelope<null>>(`${BASE}/etiquetas/entity/${entityType}/${entityId}`, { tag_id: tagId })),
  untagEntity: (entityType: string, entityId: number, tagId: number) =>
    unwrap(apiClient.delete<Envelope<null>>(`${BASE}/etiquetas/entity/${entityType}/${entityId}/${tagId}`)),

  exportContacts: () => downloadCSV(`${BASE}/export/contactos.csv`, "contactos.csv"),
  exportCompanies: () => downloadCSV(`${BASE}/export/empresas.csv`, "empresas.csv"),
  exportDeals: () => downloadCSV(`${BASE}/export/negocios.csv`, "negocios.csv"),
  exportActivities: () => downloadCSV(`${BASE}/export/actividades.csv`, "actividades.csv"),

  downloadImportTemplate: () => downloadCSV(`${BASE}/import/contactos/template.csv`, "plantilla-contactos.csv"),

  importContacts: async (file: File): Promise<ImportSummaryDto> => {
    const token = await resolveAccessToken();
    const headers: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {};
    const form = new FormData();
    form.append("file", file);
    const response = await fetch(`${apiClient.getBaseUrl()}${BASE}/import/contactos`, {
      method: "POST",
      headers,
      body: form,
      credentials: "include",
    });
    if (!response.ok) {
      let message = `Error ${response.status}`;
      try {
        const err = await response.json();
        if (typeof err?.message === "string") message = err.message;
      } catch {
        // non-JSON error body
      }
      throw new Error(message);
    }
    const body = await response.json();
    return (body?.data ?? body) as ImportSummaryDto;
  },
};
